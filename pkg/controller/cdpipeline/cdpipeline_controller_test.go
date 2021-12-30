package cdpipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	jenv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/finalizer"
)

const (
	namespace   = "stub-Namespace"
	name        = "stub-Name"
	jenkinsKind = "JenkinsFolder"
)

func emptyCdPipelineInit(t *testing.T) cdPipeApi.CDPipeline {
	t.Helper()
	return cdPipeApi.CDPipeline{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   cdPipeApi.CDPipelineSpec{},
		Status: cdPipeApi.CDPipelineStatus{},
	}
}

func (r *ReconcileCDPipeline) getJenkinsFolder(t *testing.T) *jenv1alpha1.JenkinsFolder {
	t.Helper()
	createdJenkins := &jenv1alpha1.JenkinsFolder{}
	err := r.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      fmt.Sprintf("%s-%s", name, "cd-pipeline"),
	}, createdJenkins)
	if err != nil {
		t.Fatalf("cannot find jenkins folder: %v", err)
	}
	return createdJenkins
}

func (r *ReconcileCDPipeline) getCdPipeline(t *testing.T) *cdPipeApi.CDPipeline {
	t.Helper()
	cdPipeline := &cdPipeApi.CDPipeline{}
	err := r.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cdPipeline)
	if err != nil {
		t.Fatalf("cannot find jenkins folder: %v", err)
	}
	return cdPipeline
}

func createScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(cdPipeApi.SchemeGroupVersion, &cdPipeApi.CDPipeline{}, &jenv1alpha1.JenkinsFolder{})
	return scheme
}

func TestNewReconcileCDPipeline_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	log := logr.DiscardLogger{}

	expectedReconcileCdPipeline := &ReconcileCDPipeline{
		client: client,
		scheme: scheme,
		log:    log.WithName("cd-pipeline"),
	}

	reconciledCdPipeline := NewReconcileCDPipeline(client, scheme, log)
	assert.Equal(t, expectedReconcileCdPipeline, reconciledCdPipeline)
}

func TestReconcile_Success(t *testing.T) {
	emptyCdPipeline := emptyCdPipelineInit(t)
	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&emptyCdPipeline).Build()

	reconcileCDPipeline := NewReconcileCDPipeline(client, scheme, logr.DiscardLogger{})

	_, err := reconcileCDPipeline.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}})
	assert.NoError(t, err)

	cdPipeline := reconcileCDPipeline.getCdPipeline(t)
	assert.Equal(t, cdPipeline.Status.Status, "created")

	jenkinsFolder := reconcileCDPipeline.getJenkinsFolder(t)
	assert.Equal(t, jenkinsKind, jenkinsFolder.Kind)

	assert.True(t, finalizer.ContainsString(cdPipeline.ObjectMeta.Finalizers, foregroundDeletionFinalizerName))
}

func TestReconcile_PipelineIsNotFound(t *testing.T) {
	cdPipeline := cdPipeApi.CDPipeline{}
	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&cdPipeline).Build()

	reconcileCDPipeline := NewReconcileCDPipeline(client, scheme, logr.DiscardLogger{})

	_, err := reconcileCDPipeline.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}})
	assert.NoError(t, err)
	assert.False(t, cdPipeline.Status.Available)
}

func TestReconcile_GetCdPipelineError(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconcileCDPipeline := NewReconcileCDPipeline(client, scheme, logr.DiscardLogger{})

	_, err := reconcileCDPipeline.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}})
	assert.True(t, runtime.IsNotRegisteredError(err))
}

func TestAddFinalizer_DeletionTimestampNotZero(t *testing.T) {
	var finalizerArray []string
	timeToDelete := &metav1.Time{Time: time.Now().UTC()}

	cdPipeline := cdPipeApi.CDPipeline{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			Finalizers:        finalizerArray,
			DeletionTimestamp: timeToDelete,
		},
		Spec:   cdPipeApi.CDPipelineSpec{},
		Status: cdPipeApi.CDPipelineStatus{},
	}

	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.DiscardLogger{})

	err := reconcileCdPipeline.addFinalizer(context.Background(), &cdPipeline)
	assert.NoError(t, err)

	clientCdPipeline := reconcileCdPipeline.getCdPipeline(t)
	assert.False(t, finalizer.ContainsString(clientCdPipeline.ObjectMeta.Finalizers, foregroundDeletionFinalizerName))
}

func TestAddFinalizer_DeletionTimestampIsZero(t *testing.T) {
	var finalizerArray []string

	cdPipeline := &cdPipeApi.CDPipeline{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Finalizers: finalizerArray,
		},
		Spec:   cdPipeApi.CDPipelineSpec{},
		Status: cdPipeApi.CDPipelineStatus{},
	}

	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.DiscardLogger{})

	err := reconcileCdPipeline.addFinalizer(context.Background(), cdPipeline)
	assert.NoError(t, err)

	clientCdPipeline := reconcileCdPipeline.getCdPipeline(t)
	assert.True(t, finalizer.ContainsString(clientCdPipeline.ObjectMeta.Finalizers, foregroundDeletionFinalizerName))
}

func TestSetFinishStatus_Success(t *testing.T) {
	cdPipeline := emptyCdPipelineInit(t)
	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.DiscardLogger{})

	err := reconcileCdPipeline.setFinishStatus(context.Background(), &cdPipeline)
	assert.NoError(t, err)

	clientCdPipeline := reconcileCdPipeline.getCdPipeline(t)
	assert.Equal(t, clientCdPipeline.Status.Status, "created")
}

func TestCreateJenkinsFolder_Success(t *testing.T) {
	cdPipeline := emptyCdPipelineInit(t)
	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.DiscardLogger{})

	err := reconcileCdPipeline.createJenkinsFolder(context.Background(), cdPipeline)
	assert.NoError(t, err)

	jenkinsFolder := reconcileCdPipeline.getJenkinsFolder(t)
	assert.Equal(t, jenkinsKind, jenkinsFolder.Kind)
}

func TestCreateJenkinsFolder_AlreadyExists(t *testing.T) {
	cdPipeline := emptyCdPipelineInit(t)

	jenkins := &jenv1alpha1.JenkinsFolder{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       jenkinsKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      fmt.Sprintf("%s-%s", name, "cd-pipeline"),
		},
		Status: jenv1alpha1.JenkinsFolderStatus{
			Status: "createdWithoutUsingFunction",
		},
	}

	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&cdPipeline, jenkins).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.DiscardLogger{})

	err := reconcileCdPipeline.createJenkinsFolder(context.Background(), cdPipeline)
	assert.NoError(t, err)

	createdJenkins := reconcileCdPipeline.getJenkinsFolder(t)

	assert.Equal(t, "createdWithoutUsingFunction", createdJenkins.Status.Status)
}
