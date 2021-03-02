package stage

import (
	"context"
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/factory"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"time"

	"reflect"

	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/finalizer"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	_                               reconcile.Reconciler = &ReconcileStage{}
	log                                                  = logf.Log.WithName("stage_controller")
	foregroundDeletionFinalizerName                      = "foregroundDeletion"
	envLabelDeletionFinalizer                            = "envLabelDeletion"
)

// Add creates a new Stage Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	addKnownTypes(scheme)
	return &ReconcileStage{client: mgr.GetClient(), scheme: scheme}
}

func addKnownTypes(scheme *runtime.Scheme) {
	schemeGroupVersion := schema.GroupVersion{Group: "v1.edp.epam.com", Version: "v1alpha1"}
	scheme.AddKnownTypes(schemeGroupVersion,
		&v1alpha1.EDPComponent{},
		&v1alpha1.EDPComponentList{},
	)
	metav1.AddToGroupVersion(scheme, schemeGroupVersion)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("stage-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*edpv1alpha1.Stage)
			no := e.ObjectNew.(*edpv1alpha1.Stage)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			if no.DeletionTimestamp != nil {
				return true
			}
			return false
		},
	}

	// Watch for changes to primary resource Stage
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.Stage{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	return nil
}

type ReconcileStage struct {
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileStage) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rl := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	rl.V(2).Info("reconciling Stage has been started")

	i := &edpv1alpha1.Stage{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, i); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	result, err := r.tryToDeleteCDStage(i)
	if err != nil {
		return reconcile.Result{}, err
	}
	if result != nil {
		return *result, nil
	}

	if err := r.setCDPipelineOwnerRef(i); err != nil {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, err
	}

	if err := factory.CreateDefChain(r.client, i.Spec.TriggerType).ServeRequest(i); err != nil {
		return reconcile.Result{RequeueAfter: 15 * time.Second}, err
	}

	if err := r.setFinishStatus(i); err != nil {
		return reconcile.Result{}, err
	}
	rl.V(2).Info("reconciling Stage has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileStage) tryToDeleteCDStage(stage *edpv1alpha1.Stage) (*reconcile.Result, error) {
	if stage.GetDeletionTimestamp().IsZero() {

		if !finalizer.ContainsString(stage.ObjectMeta.Finalizers, foregroundDeletionFinalizerName) {
			stage.ObjectMeta.Finalizers = append(stage.ObjectMeta.Finalizers, foregroundDeletionFinalizerName)
		}

		if stage.Spec.TriggerType == consts.AutoDeployTriggerType &&
			!finalizer.ContainsString(stage.ObjectMeta.Finalizers, envLabelDeletionFinalizer) {
			stage.ObjectMeta.Finalizers = append(stage.ObjectMeta.Finalizers, envLabelDeletionFinalizer)
		}

		if err := r.client.Update(context.TODO(), stage); err != nil {
			return &reconcile.Result{}, errors.Wrap(err, "unable to update cd stage")
		}
		return nil, nil
	}

	if err := factory.CreateDeleteChain(r.client).ServeRequest(stage); err != nil {
		return &reconcile.Result{}, err
	}

	stage.ObjectMeta.Finalizers = util.RemoveString(stage.ObjectMeta.Finalizers, envLabelDeletionFinalizer)
	if err := r.client.Update(context.TODO(), stage); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func (r *ReconcileStage) setCDPipelineOwnerRef(s *edpv1alpha1.Stage) error {
	if ow := helper.GetOwnerReference(consts.CDPipelineKind, s.GetOwnerReferences()); ow != nil {
		log.V(2).Info("CD Pipeline owner ref already exists", "name", ow.Name)
		return nil
	}
	p, err := cluster.GetCdPipeline(r.client, s.Spec.CdPipeline, s.Namespace)
	if err != nil {
		return errors.Wrapf(err, "couldn't get CD Pipeline %v from cluster", s.Spec.CdPipeline)
	}
	if err := controllerutil.SetControllerReference(p, s, r.scheme); err != nil {
		return errors.Wrapf(err, "couldn't set CD Pipeline %v owner ref", s.Spec.CdPipeline)
	}
	if err := r.client.Update(context.TODO(), s); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating stage's owner %v", s.Name)
	}
	return nil
}

func (r *ReconcileStage) setFinishStatus(s *edpv1alpha1.Stage) error {
	s.Status = edpv1alpha1.StageStatus{
		Status:          consts.FinishedStatus,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.AcceptCDStageRegistration,
		Result:          edpv1alpha1.Success,
		Value:           "active",
	}
	if err := r.client.Status().Update(context.TODO(), s); err != nil {
		if err := r.client.Update(context.TODO(), s); err != nil {
			return err
		}
	}
	return nil
}
