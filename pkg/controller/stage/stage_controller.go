package stage

import (
	"context"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/finalizer"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	jenv1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	_                               reconcile.Reconciler = &ReconcileStage{}
	log                                                  = logf.Log.WithName("stage_controller")
	foregroundDeletionFinalizerName                      = "foregroundDeletion"
)

// Add creates a new Stage Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileStage{client: mgr.GetClient(), scheme: mgr.GetScheme()}
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

	if err := r.setCDPipelineOwnerRef(i); err != nil {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, err
	}

	if err := r.tryToAddFinalizer(i); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.createJenkinsJob(*i); err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to create JenkinsJob CR")
	}

	if err := r.setFinishStatus(i); err != nil {
		return reconcile.Result{}, err
	}
	rl.V(2).Info("reconciling Stage has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileStage) setCDPipelineOwnerRef(s *edpv1alpha1.Stage) error {
	if ow := helper.GetOwnerReference(consts.CDPipelineKind, s.GetOwnerReferences()); ow != nil {
		log.V(2).Info("CD Pipeline owner ref already exists", "name", ow.Name)
		return nil
	}
	p, err := r.getCdPipeline(s.Spec.CdPipeline, s.Namespace)
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

func (r *ReconcileStage) createJenkinsJob(s edpv1alpha1.Stage) error {
	log.V(2).Info("start creating JenkinsJob CR", "name", s.Name)

	b, err := ioutil.ReadFile("/usr/local/bin/pipelines/cd-pipeline.tmpl")
	if err != nil {
		return err
	}

	jj := &jenv1alpha1.JenkinsJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "JenkinsJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v2.edp.epam.com/v1alpha1",
					Kind:               consts.StageKind,
					Name:               s.Name,
					UID:                s.UID,
					BlockOwnerDeletion: newTrue(),
				},
			},
		},
		Spec: jenv1alpha1.JenkinsJobSpec{
			StageName:     &s.Name,
			JenkinsFolder: &s.Spec.CdPipeline,
			Job: jenv1alpha1.Job{
				Name:   s.Spec.Name,
				Config: string(b),
			},
		},
	}
	if err := r.client.Create(context.TODO(), jj); err != nil {
		return errors.Wrapf(err, "couldn't create jenkins job %v", "name", jj.Name)
	}
	log.Info("JenkinsJob has been created", "name", s.Name)
	return nil
}

func newTrue() *bool {
	b := true
	return &b
}

func (r *ReconcileStage) getCdPipeline(name, namespace string) (*edpv1alpha1.CDPipeline, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	i := &edpv1alpha1.CDPipeline{}
	if err := r.client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

func (r ReconcileStage) tryToAddFinalizer(c *edpv1alpha1.Stage) error {
	if !finalizer.ContainsString(c.ObjectMeta.Finalizers, foregroundDeletionFinalizerName) {
		c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, foregroundDeletionFinalizerName)
		if err := r.client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}
