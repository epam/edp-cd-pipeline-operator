package cdpipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
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

	cdPipeline := &cdPipeApi.CDPipeline{}
	err = client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cdPipeline)
	require.NoError(t, err)
	assert.Equal(t, cdPipeline.Status.Status, "created")

	jenkinsFolder := reconcileCDPipeline.getJenkinsFolder(t)
	assert.Equal(t, jenkinsKind, jenkinsFolder.Kind)

	assert.True(t, controllerutil.ContainsFinalizer(cdPipeline, ownedStagesFinalizer))
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
	cdPipeline := cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			Finalizers:        []string{ownedStagesFinalizer},
			DeletionTimestamp: &metaV1.Time{Time: time.Now().UTC()},
		},
		Spec:   cdPipeApi.CDPipelineSpec{},
		Status: cdPipeApi.CDPipelineStatus{},
	}

	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	res, err := reconcileCdPipeline.tryToDeletePipeline(ctrl.LoggerInto(context.Background(), logr.Discard()), &cdPipeline)
	assert.NoError(t, err)
	assert.Equal(t, &reconcile.Result{}, res)

	cdPipelineProcessed := &cdPipeApi.CDPipeline{}
	err = client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cdPipelineProcessed)
	require.Error(t, err)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestAddFinalizer_PostponeDeletion(t *testing.T) {
	cdPipeline := cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			Finalizers:        []string{ownedStagesFinalizer},
			DeletionTimestamp: &metaV1.Time{Time: time.Now().UTC()},
		},
		Spec:   cdPipeApi.CDPipelineSpec{},
		Status: cdPipeApi.CDPipelineStatus{},
	}
	stage := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage",
			Namespace: namespace,
			Labels: map[string]string{
				cdPipeApi.StageCdPipelineLabelName: cdPipeline.Name,
			},
		},
	}

	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&cdPipeline, stage).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	res, err := reconcileCdPipeline.tryToDeletePipeline(ctrl.LoggerInto(context.Background(), logr.Discard()), &cdPipeline)
	assert.NoError(t, err)
	assert.Equal(t, &reconcile.Result{RequeueAfter: waitForOwnedStagesDeletion}, res)
}

func TestAddFinalizer_DeletionTimestampIsZero(t *testing.T) {
	cdPipeline := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   cdPipeApi.CDPipelineSpec{},
		Status: cdPipeApi.CDPipelineStatus{},
	}

	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	res, err := reconcileCdPipeline.tryToDeletePipeline(ctrl.LoggerInto(context.Background(), logr.Discard()), cdPipeline)
	assert.NoError(t, err)
	assert.Nil(t, res)

	cdPipelineProcessed := &cdPipeApi.CDPipeline{}
	err = client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cdPipelineProcessed)
	require.NoError(t, err)
	assert.True(t, controllerutil.ContainsFinalizer(cdPipelineProcessed, ownedStagesFinalizer))
}

func TestSetFinishStatus_Success(t *testing.T) {
	cdPipeline := emptyCdPipelineInit(t)
	scheme := createScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	reconcileCdPipeline := NewReconcileCDPipeline(client, scheme, logr.Discard())

	err := reconcileCdPipeline.setFinishStatus(context.Background(), cdPipeline)
	assert.NoError(t, err)

	cdPipelineProcessed := &cdPipeApi.CDPipeline{}
	err = client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, cdPipelineProcessed)
	require.NoError(t, err)
	assert.Equal(t, cdPipelineProcessed.Status.Status, consts.FinishedStatus)
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
