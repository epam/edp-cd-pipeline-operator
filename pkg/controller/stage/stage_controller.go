package stage

import (
	"context"
	"fmt"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/finalizer"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/types"
	"time"

	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	jenv1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	_                               reconcile.Reconciler = &ReconcileStage{}
	log                                                  = logf.Log.WithName("stage_controller")
	ForegroundDeletionFinalizerName                      = "foregroundDeletion"
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

	// Watch for changes to primary resource Stage
	err = c.Watch(&source.Kind{Type: &edpv1alpha1.Stage{}}, &handler.EnqueueRequestForObject{})
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

	if err := r.tryToAddFinalizer(i); err != nil {
		return reconcile.Result{}, err
	}

	p, err := r.getCdPipeline(i.Spec.CdPipeline, i.Namespace)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to get CD Pipeline %v", i.Spec.CdPipeline)
	}

	if !p.Status.Available {
		rl.V(2).Info("CD pipeline is not ready yet", "name", p.Name)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if err := r.createJenkinsJob(*i); err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed to create JenkinsJob CR")
	}
	rl.V(2).Info("reconciling Stage has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileStage) createJenkinsJob(s edpv1alpha1.Stage) error {
	n := fmt.Sprintf("%v-%v", s.Spec.Name, "jenkins-job")
	log.V(2).Info("start creating JenkinsJob CR", "name", n)

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
			Name:      n,
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
			StageName: &s.Name,
			Job: jenv1alpha1.Job{
				Name:   s.Spec.Name,
				Config: string(b),
			},
		},
	}
	if err := r.client.Create(context.TODO(), jj); err != nil {
		return errors.Wrapf(err, "couldn't create jenkins job %v", "name")
	}
	log.Info("JenkinsJob has been created", "name", n)
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
	if !finalizer.ContainsString(c.ObjectMeta.Finalizers, ForegroundDeletionFinalizerName) {
		c.ObjectMeta.Finalizers = append(c.ObjectMeta.Finalizers, ForegroundDeletionFinalizerName)
		if err := r.client.Update(context.TODO(), c); err != nil {
			return err
		}
	}
	return nil
}
