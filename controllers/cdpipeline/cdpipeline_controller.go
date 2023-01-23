package cdpipeline

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/finalizer"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
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

const foregroundDeletionFinalizerName = "foregroundDeletion"

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
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.V(2).Info("Reconciling CDPipeline")

	i := &cdPipeApi.CDPipeline{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get pipeline: %w", err)
	}

	if err := r.addFinalizer(ctx, i); err != nil {
		return reconcile.Result{}, err
	}

	if cluster.JenkinsEnabled(ctx, r.client, request.Namespace, log) {
		if err := r.createJenkinsFolder(ctx, i); err != nil {
			return reconcile.Result{}, err
		}
	}

	if err := r.setFinishStatus(ctx, i); err != nil {
		return reconcile.Result{}, err
	}

	log.V(2).Info("Reconciling of CD Pipeline has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileCDPipeline) addFinalizer(ctx context.Context, pipeline *cdPipeApi.CDPipeline) error {
	if !pipeline.GetDeletionTimestamp().IsZero() {
		return nil
	}

	if !finalizer.ContainsString(pipeline.ObjectMeta.Finalizers, foregroundDeletionFinalizerName) {
		pipeline.ObjectMeta.Finalizers = append(pipeline.ObjectMeta.Finalizers, foregroundDeletionFinalizerName)
	}

	if err := r.client.Update(ctx, pipeline); err != nil {
		return fmt.Errorf("failed to update pipeline: %w", err)
	}

	return nil
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

func (r *ReconcileCDPipeline) createJenkinsFolder(ctx context.Context, p *cdPipeApi.CDPipeline) error {
	jfn := fmt.Sprintf("%v-%v", p.Name, "cd-pipeline")
	log := r.log.WithValues("Jenkins folder name", jfn)
	log.V(2).Info("start creating JenkinsFolder CR", "name", jfn)

	jf := &jenkinsApi.JenkinsFolder{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "JenkinsFolder",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      jfn,
			Namespace: p.Namespace,
		},
	}
	if err := r.client.Create(ctx, jf); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			log.V(2).Info("jenkins folder cr already exists", "name", jfn)
			return nil
		}

		return fmt.Errorf("failed to reate jenkins folder %v: %w", jfn, err)
	}

	log.Info("JenkinsFolder CR has been created", "name", jfn)

	return nil
}
