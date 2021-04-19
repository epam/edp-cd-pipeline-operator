package stage

import (
	"context"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/factory"
	edpError "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"time"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/finalizer"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	specCdPipelineIndex             = "spec.cdPipeline"
	foregroundDeletionFinalizerName = "foregroundDeletion"
	envLabelDeletionFinalizer       = "envLabelDeletion"
)

type ReconcileStage struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
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
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&cdPipeApi.Stage{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileStage) AddIndex(mgr manager.Manager) error {
	log := r.Log.WithValues("field name", specCdPipelineIndex)
	log.Info("adding index for field")
	cache := mgr.GetCache()
	indexFunc := func(obj client.Object) []string {
		return []string{obj.(*cdPipeApi.Stage).Spec.CdPipeline}
	}
	if err := cache.IndexField(context.TODO(), &cdPipeApi.Stage{}, specCdPipelineIndex, indexFunc); err != nil {
		return err
	}
	log.Info("index is added")
	return nil
}

func (r *ReconcileStage) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.V(2).Info("reconciling Stage has been started")

	i := &cdPipeApi.Stage{}
	if err := r.Client.Get(ctx, request.NamespacedName, i); err != nil {
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

	if err := factory.CreateDefChain(r.Client, i.Spec.TriggerType).ServeRequest(i); err != nil {
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

		if err := r.Client.Update(ctx, stage); err != nil {
			return &reconcile.Result{}, errors.Wrap(err, "unable to update cd stage")
		}
		return nil, nil
	}

	if err := factory.CreateDeleteChain(r.Client).ServeRequest(stage); err != nil {
		return &reconcile.Result{}, err
	}

	stage.ObjectMeta.Finalizers = finalizer.RemoveString(stage.ObjectMeta.Finalizers, envLabelDeletionFinalizer)
	if err := r.Client.Update(ctx, stage); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func (r *ReconcileStage) setCDPipelineOwnerRef(ctx context.Context, s *cdPipeApi.Stage) error {
	if ow := helper.GetOwnerReference(consts.CDPipelineKind, s.GetOwnerReferences()); ow != nil {
		r.Log.V(2).Info("CD Pipeline owner ref already exists", "name", ow.Name)
		return nil
	}
	p, err := cluster.GetCdPipeline(r.Client, s.Spec.CdPipeline, s.Namespace)
	if err != nil {
		return errors.Wrapf(err, "couldn't get CD Pipeline %v from cluster", s.Spec.CdPipeline)
	}
	if err := controllerutil.SetControllerReference(p, s, r.Scheme); err != nil {
		return errors.Wrapf(err, "couldn't set CD Pipeline %v owner ref", s.Spec.CdPipeline)
	}
	if err := r.Client.Update(ctx, s); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating stage's owner %v", s.Name)
	}
	return nil
}

func (r *ReconcileStage) setFinishStatus(ctx context.Context, s *cdPipeApi.Stage) error {
	s.Status = cdPipeApi.StageStatus{
		Status:          consts.FinishedStatus,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          cdPipeApi.AcceptCDStageRegistration,
		Result:          cdPipeApi.Success,
		Value:           "active",
	}
	if err := r.Client.Status().Update(ctx, s); err != nil {
		if err := r.Client.Update(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
