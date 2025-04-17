package cluster

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
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

func TestGetCodebaseImageStreamByCodebaseBaseBranchName(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	type args struct {
		k8sCl              func(t *testing.T) client.Client
		codebaseBranchName string
	}

	tests := []struct {
		name    string
		args    args
		want    require.ValueAssertionFunc
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{
				k8sCl: func(t *testing.T) client.Client {
					obj := &codebaseApi.CodebaseImageStream{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      "test-branch",
							Namespace: "default",
							Labels: map[string]string{
								CodebaseBranchLabel: "test-branch",
							},
						},
					}

					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).Build()
				},
				codebaseBranchName: "test-branch",
			},
			want:    require.NotNil,
			wantErr: require.NoError,
		},
		{
			name: "not found",
			args: args{
				k8sCl: func(t *testing.T) client.Client {
					return fake.NewClientBuilder().WithScheme(scheme).Build()
				},
				codebaseBranchName: "non-existent-branch",
			},
			want: require.Nil,
			wantErr: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				assert.Contains(tt, err.Error(), "CodebaseImageStream not found")
			},
		},
		{
			name: "multipleFound",
			args: args{
				k8sCl: func(t *testing.T) client.Client {
					obj1 := &codebaseApi.CodebaseImageStream{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      "test-branch-1",
							Namespace: "default",
							Labels: map[string]string{
								CodebaseBranchLabel: "test-branch",
							},
						},
					}
					obj2 := &codebaseApi.CodebaseImageStream{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      "test-branch-2",
							Namespace: "default",
							Labels: map[string]string{
								CodebaseBranchLabel: "test-branch",
							},
						},
					}

					return fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj1, obj2).Build()
				},
				codebaseBranchName: "test-branch",
			},
			want: require.Nil,
			wantErr: func(tt require.TestingT, err error, i ...interface{}) {
				require.Error(tt, err)
				assert.Contains(tt, err.Error(), "multiple CodebaseImageStream found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetCodebaseImageStreamByCodebaseBaseBranchName(
				context.Background(),
				tt.args.k8sCl(t),
				tt.args.codebaseBranchName,
				"default",
			)
			tt.wantErr(t, err)
		})
	}
}
