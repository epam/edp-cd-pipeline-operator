package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

func TestDeleteRegistryViewerRbac_ServeRequest(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, rbacApi.AddToScheme(scheme))

	stage := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-stage",
		},
	}

	tests := []struct {
		name      string
		stage     *cdPipeApi.Stage
		objects   []client.Object
		prepare   func(t *testing.T)
		wantErr   assert.ErrorAssertionFunc
		wantCheck func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client)
	}{
		{
			name:  "role binding is deleted",
			stage: stage,
			objects: []client.Object{
				&rbacApi.RoleBinding{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      generateSaRegistryViewerRoleBindingName(stage),
						Namespace: "test-ns",
					},
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Openshift)
			},
			wantErr: assert.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client) {
				err := k8sClient.Get(
					context.Background(),
					client.ObjectKey{
						Name:      generateSaRegistryViewerRoleBindingName(stage),
						Namespace: stage.Namespace,
					},
					&rbacApi.RoleBinding{},
				)
				require.Error(t, err)
				assert.True(t, k8sErrors.IsNotFound(err))
			},
		},
		{
			name:  "role binding doesn't exist",
			stage: stage,
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Openshift)
			},
			wantErr: assert.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client) {
				err := k8sClient.Get(
					context.Background(),
					client.ObjectKey{
						Name:      generateSaRegistryViewerRoleBindingName(stage),
						Namespace: stage.Namespace,
					},
					&rbacApi.RoleBinding{},
				)
				require.Error(t, err)
				assert.True(t, k8sErrors.IsNotFound(err))
			},
		},
		{
			name:  "platform is not openshift",
			stage: stage,
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Kubernetes)
			},
			wantErr:   assert.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client) {},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			h := DeleteRegistryViewerRbac{
				multiClusterCl: fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build(),
			}

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage)

			tt.wantErr(t, err)
			tt.wantCheck(t, tt.stage, h.multiClusterCl)
		})
	}
}
