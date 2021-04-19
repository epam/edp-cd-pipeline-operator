package cdpipeline

import (
	"context"
	"fmt"
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/finalizer"
	jenv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

type ReconcileCDPipeline struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

const foregroundDeletionFinalizerName = "foregroundDeletion"

func (r *ReconcileCDPipeline) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*cdPipeApi.CDPipeline)
			no := e.ObjectNew.(*cdPipeApi.CDPipeline)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&cdPipeApi.CDPipeline{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileCDPipeline) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.V(2).Info("Reconciling CDPipeline")

	i := &cdPipeApi.CDPipeline{}
	if err := r.Client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if err := r.addFinalizer(ctx, i); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.createJenkinsFolder(ctx, *i); err != nil {
		return reconcile.Result{}, err
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
	if err := r.Client.Update(ctx, pipeline); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileCDPipeline) setFinishStatus(ctx context.Context, p *cdPipeApi.CDPipeline) error {
	p.Status = cdPipeApi.CDPipelineStatus{
		Status:          consts.FinishedStatus,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          cdPipeApi.SetupInitialStructureForCDPipeline,
		Result:          cdPipeApi.Success,
		Value:           "active",
	}

	if err := r.Client.Status().Update(ctx, p); err != nil {
		if err := r.Client.Update(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileCDPipeline) createJenkinsFolder(ctx context.Context, p cdPipeApi.CDPipeline) error {
	jfn := fmt.Sprintf("%v-%v", p.Name, "cd-pipeline")
	log := r.Log.WithValues("Jenkins folder name", jfn)
	log.V(2).Info("start creating JenkinsFolder CR", "name", jfn)
	jf := &jenv1alpha1.JenkinsFolder{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "JenkinsFolder",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jfn,
			Namespace: p.Namespace,
		},
	}
	if err := r.Client.Create(ctx, jf); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.V(2).Info("jenkins folder cr already exists", "name", jfn)
			return nil
		}
		return errors.Wrapf(err, "couldn't create jenkins folder %v", jfn)
	}
	log.Info("JenkinsFolder CR has been created", "name", jfn)
	return nil
}
