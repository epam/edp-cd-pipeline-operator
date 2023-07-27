package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacApi "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/externalsecrets"
)

func TestConfigureManageSecretsRBAC_ServeRequest(t *testing.T) {
	tests := []struct {
		name    string
		stage   *cdPipeApi.Stage
		objects []client.Object
		setup   func(t *testing.T)
		want    func(t *testing.T, client client.Client, stage *cdPipeApi.Stage)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "rbac is configured successfully",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "test-namespace",
				},
			},
			objects: []client.Object{
				&rbacApi.Role{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      externalSecretIntegrationRoleName,
						Namespace: "default",
					},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv(manageSecretsEnv, "true")
			},
			want: func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {
				serviceAccount := &corev1.ServiceAccount{}
				err := cl.Get(context.Background(), client.ObjectKey{
					Namespace: stage.Spec.Namespace,
					Name:      secretIntegrationServiceAccountName,
				}, serviceAccount)

				require.NoError(t, err)

				secretManagerRoleBinding := &rbacApi.RoleBinding{}
				err = cl.Get(context.Background(), client.ObjectKey{
					Namespace: stage.Namespace,
					Name:      fmt.Sprintf("eso-%s", stage.Spec.Namespace),
				}, secretManagerRoleBinding)

				require.NoError(t, err)

				secretStore := externalsecrets.NewSecretStore(secretStoreName, stage.Spec.Namespace)
				err = cl.Get(context.Background(), client.ObjectKey{
					Namespace: stage.Spec.Namespace,
					Name:      secretStoreName,
				}, secretStore)

				require.NoError(t, err)

				externalSecret := externalsecrets.NewExternalSecret(externalSecretName, stage.Spec.Namespace)
				err = cl.Get(context.Background(), client.ObjectKey{
					Namespace: stage.Spec.Namespace,
					Name:      externalSecretName,
				}, externalSecret)

				require.NoError(t, err)
			},
			wantErr: require.NoError,
		},
		{
			name: "all rbac objects already exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "test-namespace",
				},
			},
			objects: []client.Object{
				&rbacApi.Role{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      externalSecretIntegrationRoleName,
						Namespace: "default",
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      secretIntegrationServiceAccountName,
						Namespace: "test-namespace",
					},
				},
				&rbacApi.RoleBinding{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      fmt.Sprintf("eso-%s", "test-namespace"),
						Namespace: "default",
					},
				},
				externalsecrets.NewSecretStore(secretStoreName, "test-namespace"),
				externalsecrets.NewExternalSecret(externalSecretName, "test-namespace"),
			},
			setup: func(t *testing.T) {
				t.Setenv(manageSecretsEnv, "true")
			},
			want:    func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {},
			wantErr: require.NoError,
		},
		{
			name: "externalSecretIntegrationRole not found",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "test-namespace",
				},
			},
			objects: []client.Object{},
			setup: func(t *testing.T) {
				t.Setenv(manageSecretsEnv, "true")
			},
			want: func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get external-secret-integration role")
			},
		},
		{
			name: "env variable manageSecrets is not set",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "test-namespace",
				},
			},
			objects: []client.Object{},
			setup:   func(t *testing.T) {},
			want:    func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {},
			wantErr: require.NoError,
		},
		{
			name: "env variable manageSecrets is invalid",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "test-namespace",
				},
			},
			objects: []client.Object{},
			setup: func(t *testing.T) {
				t.Setenv(manageSecretsEnv, "invalid value")
			},
			want:    func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			sc := runtime.NewScheme()

			require.NoError(t, cdPipeApi.AddToScheme(sc))
			require.NoError(t, corev1.AddToScheme(sc))
			require.NoError(t, rbacApi.AddToScheme(sc))

			h := ConfigureManageSecretsRBAC{
				client: fake.NewClientBuilder().
					WithScheme(sc).
					WithObjects(tt.objects...).
					Build(),
				log: logr.Discard(),
			}

			tt.wantErr(t, h.ServeRequest(tt.stage))
			tt.want(t, h.client, tt.stage)
		})
	}
}
