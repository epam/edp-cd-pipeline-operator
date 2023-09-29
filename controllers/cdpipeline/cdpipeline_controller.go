package cdpipeline

import (
	"context"
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
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

func NewReconcileCDPipeline(c client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCDPipeline {
	return &ReconcileCDPipeline{
		client: c,
		scheme: scheme,
		log:    log.WithName("cd-pipeline"),
	}
}

type ReconcileCDPipeline struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

const (
	ownedStagesFinalizer       = "edp.epam.com/ownedStages"
	waitForOwnedStagesDeletion = time.Second * 2
)

func (r *ReconcileCDPipeline) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo, ok := e.ObjectOld.(*cdPipeApi.CDPipeline)
			if !ok {
				return false
			}
			no, ok := e.ObjectNew.(*cdPipeApi.CDPipeline)
			if !ok {
				return false
			}

			if no.DeletionTimestamp != nil {
				return true
			}

			return !reflect.DeepEqual(oo.Spec, no.Spec)
		},
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&cdPipeApi.CDPipeline{}, builder.WithPredicates(p)).
		Complete(r); err != nil {
		return fmt.Errorf("failed to create controller manager: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=cdpipelines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=cdpipelines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=v2.edp.epam.com,namespace=placeholder,resources=cdpipelines/finalizers,verbs=update

func (r *ReconcileCDPipeline) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling CDPipeline")

	pipeline := &cdPipeApi.CDPipeline{}
	if err := r.client.Get(ctx, request.NamespacedName, pipeline); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get pipeline: %w", err)
	}

	if err := r.applyDefaults(ctx, pipeline); err != nil {
		return reconcile.Result{}, err
	}

	result, err := r.tryToDeletePipeline(ctx, pipeline)
	if err != nil {
		return reconcile.Result{}, err
	}

	if result != nil {
		return *result, nil
	}

	if err := r.setFinishStatus(ctx, pipeline); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Reconciling of CD Pipeline has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileCDPipeline) tryToDeletePipeline(ctx context.Context, pipeline *cdPipeApi.CDPipeline) (*reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if pipeline.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(pipeline, ownedStagesFinalizer) {
			if err := r.client.Update(ctx, pipeline); err != nil {
				return &reconcile.Result{}, fmt.Errorf("failed to update pipeline: %w", err)
			}

			log.Info("Finalizer has been added to CDPipeline", "finalizer", ownedStagesFinalizer)
		}

		return nil, nil
	}

	log.Info("Deleting CDPipeline")

	hasStages, err := r.hasActiveOwnedStages(ctx, pipeline)
	if err != nil {
		return &reconcile.Result{}, err
	}

	// if pipeline has active stages, postpone deletion
	// because if we delete pipeline before stages,
	// stages deletion chain will be broken
	if hasStages {
		log.Info("Deleting stages of CDPipeline")

		if err := r.client.DeleteAllOf(
			ctx,
			&cdPipeApi.Stage{},
			client.InNamespace(pipeline.Namespace),
			client.MatchingLabels(map[string]string{cdPipeApi.StageCdPipelineLabelName: pipeline.Name}),
		); err != nil {
			return &reconcile.Result{}, fmt.Errorf("failed to delete stages: %w", err)
		}

		log.Info("CDPipeline has active stages. Postpone deletion")

		return &reconcile.Result{RequeueAfter: waitForOwnedStagesDeletion}, nil
	}

	log.Info("Removing finalizer from CDPipeline", "finalizer", ownedStagesFinalizer)

	controllerutil.RemoveFinalizer(pipeline, ownedStagesFinalizer)

	if err := r.client.Update(ctx, pipeline); err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to update pipeline: %w", err)
	}

	log.Info("CDPipeline has been deleted")

	return &reconcile.Result{}, nil
}

func (r *ReconcileCDPipeline) setFinishStatus(ctx context.Context, p *cdPipeApi.CDPipeline) error {
	p.Status = cdPipeApi.CDPipelineStatus{
		Status:          consts.FinishedStatus,
		Available:       true,
		LastTimeUpdated: metaV1.Now(),
		Username:        "system",
		Action:          cdPipeApi.SetupInitialStructureForCDPipeline,
		Result:          cdPipeApi.Success,
		Value:           "active",
	}

	if err := r.client.Status().Update(ctx, p); err != nil {
		if err = r.client.Update(ctx, p); err != nil {
			return fmt.Errorf("failed to update pipeline status: %w", err)
		}
	}

	return nil
}

// hasActiveOwnedStages checks if there are any active stages owned by the pipeline.
func (r *ReconcileCDPipeline) hasActiveOwnedStages(ctx context.Context, pipeline *cdPipeApi.CDPipeline) (bool, error) {
	stages := &cdPipeApi.StageList{}
	if err := r.client.List(
		ctx,
		stages,
		client.InNamespace(pipeline.Namespace), client.MatchingLabels{cdPipeApi.StageCdPipelineLabelName: pipeline.Name},
	); err != nil {
		return false, fmt.Errorf("failed to list stages: %w", err)
	}

	return len(stages.Items) > 0, nil
}

func (r *ReconcileCDPipeline) applyDefaults(ctx context.Context, pipeline *cdPipeApi.CDPipeline) error {
	if pipeline.Spec.ApplicationsToPromote == nil {
		// currently it is not possible to set default as empty slice in the CRD definition by controller-gen
		pipeline.Spec.ApplicationsToPromote = []string{}
		if err := r.client.Update(ctx, pipeline); err != nil {
			return fmt.Errorf("failed to update pipeline: %w", err)
		}
	}

	return nil
}
