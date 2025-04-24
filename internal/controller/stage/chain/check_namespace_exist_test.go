package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

func TestCheckNamespaceExist_ServeRequest(t *testing.T) {
	scheme := runtime.NewScheme()

	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, projectApi.AddToScheme(scheme))

	tests := []struct {
		name    string
		stage   *cdPipeApi.Stage
		prepare func(t *testing.T)
		objects []client.Object
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "namespace exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "stage-1-ns",
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Kubernetes)
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "stage-1-ns",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "namespace doesn't exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "stage-1-ns",
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Kubernetes)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "doesn't exist")
			},
		},
		{
			name: "project exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "stage-1-ns",
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Openshift)
			},
			objects: []client.Object{
				&projectApi.Project{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "stage-1-ns",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "project doesn't exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "stage-1-ns",
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Openshift)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "doesn't exist")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			h := CheckNamespaceExist{
				client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build(),
			}

			tt.wantErr(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage))
		})
	}
}
