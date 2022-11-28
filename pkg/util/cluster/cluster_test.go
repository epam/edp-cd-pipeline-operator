package cluster

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

const (
	name          = "stub-name"
	namespace     = "stub-namespace"
	isDebugModeOn = "true"
)

func TestGetCdPipeline_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.CDPipeline{})

	cdPipeline := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	_, err := GetCdPipeline(c, name, namespace)
	assert.NoError(t, err)
}

func TestGetCdPipeline_IsNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.CDPipeline{})
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	_, err := GetCdPipeline(c, name, namespace)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestGetCodebaseImageStream_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &codebaseApi.CodebaseImageStream{})

	cdPipeline := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline).Build()

	_, err := GetCodebaseImageStream(c, name, namespace)
	assert.NoError(t, err)
}

func TestGetCodebaseImageStream_isNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &codebaseApi.CodebaseImageStream{})
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	_, err := GetCodebaseImageStream(c, name, namespace)
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

func TestJenkinsEnabled(t *testing.T) {
	scheme := runtime.NewScheme()
	err := jenkinsApi.AddToScheme(scheme)
	require.NoError(t, err)

	type args struct {
		k8sObjects []client.Object
		namespace  string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "jenkins is enabled",
			args: args{
				k8sObjects: []client.Object{
					&jenkinsApi.Jenkins{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      "jenkins",
							Namespace: "default",
						},
					},
				},
				namespace: "default",
			},
			want: true,
		},
		{
			name: "jenkins is disabled",
			args: args{
				k8sObjects: []client.Object{},
				namespace:  "default",
			},
			want: false,
		},
		{
			name: "jenkins is in another namespace",
			args: args{
				k8sObjects: []client.Object{
					&jenkinsApi.Jenkins{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      "jenkins",
							Namespace: "test-namespace",
						},
					},
				},
				namespace: "default",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.args.k8sObjects...).Build()
			got := JenkinsEnabled(context.Background(), fakeClient, tt.args.namespace, logr.Discard())
			assert.Equal(t, tt.want, got)
		})
	}
}
