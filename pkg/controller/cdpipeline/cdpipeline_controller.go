package cdpipeline

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/consts"
	jenv1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var (
	_   reconcile.Reconciler = &ReconcileCDPipeline{}
	log                      = logf.Log.WithName("cd_pipeline_controller")
)

// Add creates a new CDPipeline Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCDPipeline{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cdpipeline-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*edpv1alpha1.CDPipeline)
			no := e.ObjectNew.(*edpv1alpha1.CDPipeline)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			return false
		},
	}

	// Watch for changes to primary resource CDPipeline
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.CDPipeline{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	return nil
}

// ReconcileCDPipeline reconciles a CDPipeline object
type ReconcileCDPipeline struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileCDPipeline) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rlog := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	rlog.V(2).Info("Reconciling CDPipeline")

	i := &edpv1alpha1.CDPipeline{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if err := r.createJenkinsFolder(*i); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.setFinishStatus(i); err != nil {
		return reconcile.Result{}, err
	}
	rlog.V(2).Info("Reconciling of CD Pipeline has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCDPipeline) setFinishStatus(p *edpv1alpha1.CDPipeline) error {
	p.Status = edpv1alpha1.CDPipelineStatus{
		Status:          consts.FinishedStatus,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.SetupInitialStructureForCDPipeline,
		Result:          edpv1alpha1.Success,
		Value:           "active",
	}

	if err := r.client.Status().Update(context.TODO(), p); err != nil {
		if err := r.client.Update(context.TODO(), p); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileCDPipeline) createJenkinsFolder(p edpv1alpha1.CDPipeline) error {
	jfn := fmt.Sprintf("%v-%v", p.Name, "cd-pipeline")
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
	if err := r.client.Create(context.TODO(), jf); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.V(2).Info("jenkins folder cr already exists", "name", jfn)
			return nil
		}
		return errors.Wrapf(err, "couldn't create jenkins folder %v", jfn)
	}
	log.Info("JenkinsFolder CR has been created", "name", jfn)
	return nil
}
