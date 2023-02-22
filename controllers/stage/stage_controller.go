package stage

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain"
	edpError "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/objectmodifier"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/finalizer"
)

const (
	foregroundDeletionFinalizerName = "foregroundDeletion"
	envLabelDeletionFinalizer       = "envLabelDeletion"
	const15Requeue                  = 15 * time.Second
	nameLogKey                      = "name"
)

func NewReconcileStage(
	c client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	stageModifier objectmodifier.StageModifier,
) *ReconcileStage {
	return &ReconcileStage{
		client:        c,
		scheme:        scheme,
		log:           log.WithName("cd-stage"),
		stageModifier: stageModifier,
	}
}

type ReconcileStage struct {
	client        client.Client
	scheme        *runtime.Scheme
	log           logr.Logger
	stageModifier objectmodifier.StageModifier
}

func (r *ReconcileStage) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo, ok := e.ObjectOld.(*cdPipeApi.Stage)
			if !ok {
				return false
			}
			no, ok := e.ObjectNew.(*cdPipeApi.Stage)
			if !ok {
				return false
			}
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
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&cdPipeApi.Stage{}, builder.WithPredicates(p)).
		Watches(&source.Kind{Type: &cdPipeApi.CDPipeline{}}, NewPipelineEventHandler(r.client, r.log)).
		Complete(r); err != nil {
		return fmt.Errorf("failed to create controller manager: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=stages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=stages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=stages/finalizers,verbs=update

func (r *ReconcileStage) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.V(2).Info("reconciling Stage has been started")

	stage := &cdPipeApi.Stage{}
	if err := r.client.Get(ctx, request.NamespacedName, stage); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get namespace: %w", err)
	}

	patched, err := r.stageModifier.Apply(ctrl.LoggerInto(ctx, log), stage)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to apply stage changes: %w", err)
	}

	if patched {
		log.Info("Stage default values has been patched")

		return reconcile.Result{}, nil
	}

	result, err := r.tryToDeleteCDStage(ctx, stage)
	if err != nil {
		return reconcile.Result{}, err
	}

	if result != nil {
		return *result, nil
	}

	if err = chain.CreateChain(ctx, r.client, request.Namespace, stage.Spec.TriggerType).ServeRequest(stage); err != nil {
		var e edpError.CISNotFoundError
		if errors.As(err, &e) {
			log.Error(err, "cis wasn't found. reconcile again...")
			return reconcile.Result{RequeueAfter: const15Requeue}, nil
		}

		if statusErr := r.setFailedStatus(ctx, stage, err); err != nil {
			return reconcile.Result{}, statusErr
		}

		return reconcile.Result{RequeueAfter: const15Requeue}, fmt.Errorf("failed to handle the chain: %w", err)
	}

	if err := r.setFinishStatus(ctx, stage); err != nil {
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
			return &reconcile.Result{}, fmt.Errorf("failed to update cd stage: %w", err)
		}

		return nil, nil
	}

	if err := chain.CreateDeleteChain(ctx, r.client, stage.Namespace).ServeRequest(stage); err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to delete chain: %w", err)
	}

	stage.ObjectMeta.Finalizers = finalizer.RemoveString(stage.ObjectMeta.Finalizers, envLabelDeletionFinalizer)
	if err := r.client.Update(ctx, stage); err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to update cd pipeline stage: %w", err)
	}

	return &reconcile.Result{}, nil
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
		if err = r.client.Update(ctx, s); err != nil {
			return fmt.Errorf("failed to update stage status: %w", err)
		}
	}

	return nil
}

func (r *ReconcileStage) setFailedStatus(ctx context.Context, stage *cdPipeApi.Stage, err error) error {
	stage.Status = cdPipeApi.StageStatus{
		Status:          consts.FailedStatus,
		Available:       false,
		LastTimeUpdated: metaV1.Now(),
		Username:        stage.Status.Username,
		Result:          cdPipeApi.Error,
		DetailedMessage: err.Error(),
		Value:           consts.FailedStatus,
	}

	if err = r.client.Status().Update(ctx, stage); err != nil {
		return fmt.Errorf("failed to update stage status: %w", err)
	}

	r.log.Info("Stage status has been updated.", nameLogKey, stage.Name)

	return nil
}
