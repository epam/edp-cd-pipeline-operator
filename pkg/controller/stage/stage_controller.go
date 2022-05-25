package stage

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain"
	edpError "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/finalizer"
)

const (
	foregroundDeletionFinalizerName = "foregroundDeletion"
	envLabelDeletionFinalizer       = "envLabelDeletion"
)

func NewReconcileStage(client client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileStage {
	return &ReconcileStage{
		client: client,
		scheme: scheme,
		log:    log.WithName("cd-stage"),
	}
}

type ReconcileStage struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

func (r *ReconcileStage) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*cdPipeApi.Stage)
			no := e.ObjectNew.(*cdPipeApi.Stage)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			if no.DeletionTimestamp != nil {
				return true
			}
			if no.Status.ShouldBeHandled {
				return true
			}
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&cdPipeApi.Stage{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileStage) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.V(2).Info("reconciling Stage has been started")

	i := &cdPipeApi.Stage{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	result, err := r.tryToDeleteCDStage(ctx, i)
	if err != nil {
		return reconcile.Result{}, err
	}
	if result != nil {
		return *result, nil
	}

	if err := r.setCDPipelineOwnerRef(ctx, i); err != nil {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, err
	}

	if err := chain.CreateChain(r.client, i.Spec.TriggerType).ServeRequest(i); err != nil {
		switch errors.Cause(err).(type) {
		case edpError.CISNotFound:
			log.Error(err, "cis wasn't found. reconcile again...")
			return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
		default:
			return reconcile.Result{RequeueAfter: 15 * time.Second}, err
		}
	}

	if err := r.setFinishStatus(ctx, i); err != nil {
		return reconcile.Result{}, err
	}
	log.V(2).Info("reconciling Stage has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileStage) tryToDeleteCDStage(ctx context.Context, stage *cdPipeApi.Stage) (*reconcile.Result, error) {
	if stage.GetDeletionTimestamp().IsZero() {

		if !finalizer.ContainsString(stage.ObjectMeta.Finalizers, foregroundDeletionFinalizerName) {
			stage.ObjectMeta.Finalizers = append(stage.ObjectMeta.Finalizers, foregroundDeletionFinalizerName)
		}

		if stage.Spec.TriggerType == consts.AutoDeployTriggerType &&
			!finalizer.ContainsString(stage.ObjectMeta.Finalizers, envLabelDeletionFinalizer) {
			stage.ObjectMeta.Finalizers = append(stage.ObjectMeta.Finalizers, envLabelDeletionFinalizer)
		}

		if err := r.client.Update(ctx, stage); err != nil {
			return &reconcile.Result{}, errors.Wrap(err, "unable to update cd stage")
		}
		return nil, nil
	}

	if err := chain.CreateDeleteChain(r.client).ServeRequest(stage); err != nil {
		return &reconcile.Result{}, err
	}

	stage.ObjectMeta.Finalizers = finalizer.RemoveString(stage.ObjectMeta.Finalizers, envLabelDeletionFinalizer)
	if err := r.client.Update(ctx, stage); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func (r *ReconcileStage) setCDPipelineOwnerRef(ctx context.Context, s *cdPipeApi.Stage) error {
	if ow := helper.GetOwnerReference(consts.CDPipelineKind, s.GetOwnerReferences()); ow != nil {
		r.log.V(2).Info("CD Pipeline owner ref already exists", "name", ow.Name)
		return nil
	}
	p, err := cluster.GetCdPipeline(r.client, s.Spec.CdPipeline, s.Namespace)
	if err != nil {
		return fmt.Errorf("couldn't get CD Pipeline %s from cluster: %w", s.Spec.CdPipeline, err)
	}
	if err := controllerutil.SetControllerReference(p, s, r.scheme); err != nil {
		return fmt.Errorf("couldn't set CD Pipeline %s owner ref: %w", s.Spec.CdPipeline, err)
	}
	if err := r.client.Update(ctx, s); err != nil {
		return fmt.Errorf("an error has been occurred while updating stage's owner %s: %w", s.Name, err)
	}
	return nil
}

func (r *ReconcileStage) setFinishStatus(ctx context.Context, s *cdPipeApi.Stage) error {
	s.Status = cdPipeApi.StageStatus{
		Status:          consts.FinishedStatus,
		Available:       true,
		LastTimeUpdated: metaV1.Now(),
		Username:        "system",
		Action:          cdPipeApi.AcceptCDStageRegistration,
		Result:          cdPipeApi.Success,
		Value:           "active",
		ShouldBeHandled: false,
	}
	if err := r.client.Status().Update(ctx, s); err != nil {
		if err := r.client.Update(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
