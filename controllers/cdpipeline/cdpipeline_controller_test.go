package cdpipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/finalizer"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

const (
	namespace   = "stub-Namespace"
	name        = "stub-Name"
	jenkinsKind = "JenkinsFolder"
)

func emptyCdPipelineInit(t *testing.T) *cdPipeApi.CDPipeline {
	t.Helper()

	return &cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   cdPipeApi.CDPipelineSpec{},
		Status: cdPipeApi.CDPipelineStatus{},
	}
}

func (r *ReconcileCDPipeline) getJenkinsFolder(t *testing.T) *jenkinsApi.JenkinsFolder {
	t.Helper()

	createdJenkins := &jenkinsApi.JenkinsFolder{}
	if err := r.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      fmt.Sprintf("%s-%s", name, "cd-pipeline"),
	}, createdJenkins); err != nil {
		t.Fatalf("cannot find jenkins folder: %v", err)
	}

	return createdJenkins
}

func (r *ReconcileCDPipeline) getCdPipeline(t *testing.T) *cdPipeApi.CDPipeline {
	t.Helper()

	cdPipeline := &cdPipeApi.CDPipeline{}
	if err := r.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cdPipeline); err != nil {
		t.Fatalf("cannot find jenkins folder: %v", err)
	}

	return cdPipeline
}

func createScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()

	err := cdPipeApi.AddToScheme(scheme)
	require.NoError(t, err)

	err = jenkinsApi.AddToScheme(scheme)
	require.NoError(t, err)

	return scheme
}

func TestNewReconcileCDPipeline_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	log := logr.Discard()

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
	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: namespace,
			Name:      "stub-Jenkins",
		},
	}
	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(emptyCdPipeline, jenkins).Build()

	reconcileCDPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

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

	reconcileCDPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

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

	reconcileCDPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	_, err := reconcileCDPipeline.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}})
	assert.Contains(t, err.Error(), "no kind is registered")
}

func TestAddFinalizer_DeletionTimestampNotZero(t *testing.T) {
	var finalizerArray []string

	timeToDelete := &metaV1.Time{Time: time.Now().UTC()}

	cdPipeline := cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
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

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	err := reconcileCdPipeline.addFinalizer(context.Background(), &cdPipeline)
	assert.NoError(t, err)

	clientCdPipeline := reconcileCdPipeline.getCdPipeline(t)
	assert.False(t, finalizer.ContainsString(clientCdPipeline.ObjectMeta.Finalizers, foregroundDeletionFinalizerName))
}

func TestAddFinalizer_DeletionTimestampIsZero(t *testing.T) {
	var finalizerArray []string

	cdPipeline := &cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Finalizers: finalizerArray,
		},
		Spec:   cdPipeApi.CDPipelineSpec{},
		Status: cdPipeApi.CDPipelineStatus{},
	}

	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	err := reconcileCdPipeline.addFinalizer(context.Background(), cdPipeline)
	assert.NoError(t, err)

	clientCdPipeline := reconcileCdPipeline.getCdPipeline(t)
	assert.True(t, finalizer.ContainsString(clientCdPipeline.ObjectMeta.Finalizers, foregroundDeletionFinalizerName))
}

func TestSetFinishStatus_Success(t *testing.T) {
	cdPipeline := emptyCdPipelineInit(t)
	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	err := reconcileCdPipeline.setFinishStatus(context.Background(), cdPipeline)
	assert.NoError(t, err)

	clientCdPipeline := reconcileCdPipeline.getCdPipeline(t)
	assert.Equal(t, clientCdPipeline.Status.Status, "created")
}

func TestCreateJenkinsFolder_Success(t *testing.T) {
	cdPipeline := emptyCdPipelineInit(t)
	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	err := reconcileCdPipeline.createJenkinsFolder(context.Background(), cdPipeline)
	assert.NoError(t, err)

	jenkinsFolder := reconcileCdPipeline.getJenkinsFolder(t)
	assert.Equal(t, jenkinsKind, jenkinsFolder.Kind)
}

func TestCreateJenkinsFolder_AlreadyExists(t *testing.T) {
	cdPipeline := emptyCdPipelineInit(t)

	jenkins := &jenkinsApi.JenkinsFolder{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       jenkinsKind,
		},
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: namespace,
			Name:      fmt.Sprintf("%s-%s", name, "cd-pipeline"),
		},
		Status: jenkinsApi.JenkinsFolderStatus{
			Status: "createdWithoutUsingFunction",
		},
	}

	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline, jenkins).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	err := reconcileCdPipeline.createJenkinsFolder(context.Background(), cdPipeline)
	assert.NoError(t, err)

	createdJenkins := reconcileCdPipeline.getJenkinsFolder(t)

	assert.Equal(t, "createdWithoutUsingFunction", createdJenkins.Status.Status)
}
