package cdpipeline

import (
	"context"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/service/cdpipeline"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

	// Watch for changes to primary resource CDPipeline
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.CDPipeline{}}, &handler.EnqueueRequestForObject{})
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

// Reconcile reads that state of the cluster for a CDPipeline object and makes changes based on the state read
// and what is in the CDPipeline.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCDPipeline) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLog := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLog.Info("Reconciling CDPipeline")
	// Fetch the CDPipeline instance
	instance := &edpv1alpha1.CDPipeline{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	reqLog.Info("Successful fetching of CR", "CDPipeline", *instance)

	defer r.updateStatus(instance)

	p, err := platform.NewPlatformService(helper.GetPlatformTypeEnv())
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "Failed to create Platform service")
	}
	pipelineService := cdpipeline.CDPipelineService{
		Resource: instance,
		Client:   r.client,
		Platform: p,
	}
	err = pipelineService.CreateCDPipeline()
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "Failed to create CD Pipeline")
	}

	reqLog.Info("Reconciling of CD Pipeline has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCDPipeline) updateStatus(instance *edpv1alpha1.CDPipeline) {
	err := r.client.Status().Update(context.TODO(), instance)
	if err != nil {
		_ = r.client.Update(context.TODO(), instance)
	}
}
