package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	rbacApi "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
)

func TestConfigureTenantAdminRbac_ServeRequest(t *testing.T) {
	t.Parallel()

	const namespace = "test-ns"

	scheme := runtime.NewScheme()
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, rbacApi.AddToScheme(scheme))

	tests := []struct {
		name      string
		stage     *cdPipeApi.Stage
		objects   []runtime.Object
		wantErr   require.ErrorAssertionFunc
		wantCheck func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client)
	}{
		{
			name: "rbac configuration is successful",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-stage",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "stage-1-ns",
				},
			},
			wantErr: require.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client) {
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      tenantAdminRbName,
					Namespace: stage.Spec.Namespace,
				}, &rbacApi.RoleBinding{}))
			},
		},
		{
			name: "rbac already exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-stage",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "stage-1-ns",
				},
			},
			objects: []runtime.Object{
				&rbacApi.RoleBinding{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      tenantAdminRbName,
						Namespace: "stage-1-ns",
					},
				},
			},
			wantErr: require.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client) {
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      tenantAdminRbName,
					Namespace: stage.Spec.Namespace,
				}, &rbacApi.RoleBinding{}))
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()

			h := ConfigureTenantAdminRbac{
				rbac: rbac.NewRbacManager(k8sClient, logr.Discard()),
			}

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage)
			tt.wantErr(t, err)
			tt.wantCheck(t, tt.stage, k8sClient)
		})
	}
}
