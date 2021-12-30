package cluster

import (
	"fmt"
	"os"
	"testing"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	k8sApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
)

const (
	name          = "stub-name"
	namespace     = "stub-namespace"
	isDebugModeOn = "true"
)

func TestGetCdPipeline_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &v1alpha1.CDPipeline{})

	cdPipeline := &v1alpha1.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	_, err := GetCdPipeline(client, name, namespace)
	assert.NoError(t, err)
}

func TestGetCdPipeline_IsNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &v1alpha1.CDPipeline{})
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	_, err := GetCdPipeline(client, name, namespace)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestGetCodebaseImageStream_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &codebaseApi.CodebaseImageStream{})

	cdPipeline := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	_, err := GetCodebaseImageStream(client, name, namespace)
	assert.NoError(t, err)
}

func TestGetCodebaseImageStream_isNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &codebaseApi.CodebaseImageStream{})
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	_, err := GetCodebaseImageStream(client, name, namespace)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestGetWatchNamespace_Success(t *testing.T) {
	err := os.Setenv(watchNamespaceEnvVar, namespace)
	if err != nil {
		t.Fatalf("cannot set env variable: %v", err)
	}
	defer func() {
		err = os.Unsetenv(watchNamespaceEnvVar)
		if err != nil {
			t.Fatalf("cannot unset env variable: %v", err)
		}
	}()

	envNamespace, err := GetWatchNamespace()
	assert.NoError(t, err)
	assert.Equal(t, namespace, envNamespace)
}

func TestGetWatchNamespace_IsNotSet(t *testing.T) {
	watchNamespace, err := GetWatchNamespace()
	assert.Equal(t, fmt.Errorf("%s must be set", watchNamespaceEnvVar), err)
	assert.Empty(t, watchNamespace)
}

func TestGetDebugMode_Success(t *testing.T) {
	err := os.Setenv(debugModeEnvVar, isDebugModeOn)
	if err != nil {
		t.Fatalf("cannot set env variable: %v", err)
	}
	defer func() {
		err = os.Unsetenv(debugModeEnvVar)
		if err != nil {
			t.Fatalf("cannot unset env variable: %v", err)
		}
	}()

	debugMode, err := GetDebugMode()
	assert.NoError(t, err)
	assert.True(t, debugMode)
}

func TestGetDebugMode_CantRead(t *testing.T) {
	err := os.Setenv(debugModeEnvVar, "")
	if err != nil {
		t.Fatalf("cannot set env variable: %v", err)
	}
	defer func() {
		err = os.Unsetenv(debugModeEnvVar)
		if err != nil {
			t.Fatalf("cannot unset env variable: %v", err)
		}
	}()

	debugMode, err := GetDebugMode()
	assert.False(t, debugMode)
}

func TestGetDebugMode_IsNotSet(t *testing.T) {
	debugMode, err := GetDebugMode()
	assert.NoError(t, err)
	assert.False(t, debugMode)
}
