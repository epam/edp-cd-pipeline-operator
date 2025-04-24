package stage

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain"
	edpError "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/objectmodifier"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

const (
	envLabelDeletionFinalizer   = "envLabelDeletion"
	const15Requeue              = 15 * time.Second
	waitForParentStagesDeletion = time.Second
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
		Watches(&cdPipeApi.CDPipeline{}, NewPipelineEventHandler(r.client, r.log)).
		Complete(r); err != nil {
		return fmt.Errorf("failed to create controller manager: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=stages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=stages/finalizers,verbs=update
// +kubebuilder:rbac:groups=argoproj.io,namespace=placeholder,resources=applicationsets,verbs=get;list;watch;update;patch;create

func (r *ReconcileStage) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling Stage has been started")

	stage := &cdPipeApi.Stage{}
	if err := r.client.Get(ctx, request.NamespacedName, stage); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("Stage has been deleted")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get stage: %w", err)
	}

	patched, err := r.stageModifier.Apply(ctx, stage)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to apply stage changes: %w", err)
	}

	if patched {
		log.Info("Stage default values has been patched")
	}

	result, err := r.tryToDeleteCDStage(ctx, stage)
	if err != nil {
		return reconcile.Result{}, err
	}

	if result != nil {
		return *result, nil
	}

	ch, err := chain.CreateChain(ctx, r.client, stage)
	if err != nil {
		if statusErr := r.setFailedStatus(ctx, stage, err); statusErr != nil {
			log.Error(statusErr, "Failed to set failed status")
		}

		return reconcile.Result{}, fmt.Errorf("failed to create chain: %w", err)
	}

	if err = ch.ServeRequest(ctx, stage); err != nil {
		var e edpError.CISNotFoundError
		if errors.As(err, &e) {
			log.Error(err, "cis wasn't found. reconcile again...")
			return reconcile.Result{RequeueAfter: const15Requeue}, nil
		}

		if statusErr := r.setFailedStatus(ctx, stage, err); statusErr != nil {
			return reconcile.Result{}, statusErr
		}

		return reconcile.Result{RequeueAfter: const15Requeue}, fmt.Errorf("failed to handle the chain: %w", err)
	}

	if err := r.setFinishStatus(ctx, stage); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Reconciling Stage has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileStage) tryToDeleteCDStage(ctx context.Context, stage *cdPipeApi.Stage) (*reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if stage.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(stage, envLabelDeletionFinalizer) {
			if err := r.client.Update(ctx, stage); err != nil {
				return &reconcile.Result{}, fmt.Errorf("failed to update cd stage: %w", err)
			}

			log.Info("Finalizer has been added to Stage", "finalizer", envLabelDeletionFinalizer)
		}

		return nil, nil
	}

	log.Info("Deleting Stage")

	isLastStage, err := r.isLastStage(ctx, stage)
	if err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to check if stage is last: %w", err)
	}

	// if stage is not last, we should postpone deletion
	// and wait for all parent stages to be deleted
	// because in the chain we get previous stage
	if !isLastStage {
		log.Info("Stage is not last. Postpone deletion")

		return &reconcile.Result{RequeueAfter: waitForParentStagesDeletion}, nil
	}

	log.Info("Stage is last. Delete chain")

	ch, err := chain.CreateDeleteChain(ctx, r.client, stage)
	if err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to create delete chain: %w", err)
	}

	if err = ch.ServeRequest(ctx, stage); err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to delete Stage: %w", err)
	}

	log.Info("Removing finalizer from Stage", "finalizer", envLabelDeletionFinalizer)

	controllerutil.RemoveFinalizer(stage, envLabelDeletionFinalizer)

	if err := r.client.Update(ctx, stage); err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to update cd pipeline stage: %w", err)
	}

	log.Info("Stage has been deleted")

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
	log := ctrl.LoggerFrom(ctx)

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

	log.Info("Stage failed status has been updated")

	return nil
}

// isLastStage checks if stage is last in the pipeline.
func (r *ReconcileStage) isLastStage(ctx context.Context, stage *cdPipeApi.Stage) (bool, error) {
	stages := &cdPipeApi.StageList{}
	if err := r.client.List(
		ctx,
		stages,
		client.InNamespace(stage.Namespace),
		client.MatchingLabels{cdPipeApi.StageCdPipelineLabelName: stage.Spec.CdPipeline},
	); err != nil {
		return false, fmt.Errorf("failed to get stages: %w", err)
	}

	for i := range stages.Items {
		if stages.Items[i].Name == stage.Name {
			continue
		}

		if stages.Items[i].Spec.Order > stage.Spec.Order {
			return false, nil
		}
	}

	return true, nil
}
