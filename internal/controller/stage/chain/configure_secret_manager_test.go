package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/esotest"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/externalsecrets"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

func TestConfigureManageSecretsRBAC_ServeRequest(t *testing.T) {
	tests := []struct {
		name        string
		stage       *cdPipeApi.Stage
		objects     []client.Object
		esoVersions []string
		setup       func(t *testing.T)
		want        func(t *testing.T, client client.Client, stage *cdPipeApi.Stage)
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "eso is configured successfully",
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
			esoVersions: []string{"v1"},
			setup: func(t *testing.T) {
				t.Setenv(platform.SecretManagerEnv, platform.SecretManagerESO)
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
					Name:      secretManagerRoleBindingName(stage.Spec.Namespace),
				}, secretManagerRoleBinding)

				require.NoError(t, err)
				require.Equal(t, rbacApi.RoleRef{
					APIGroup: rbacApi.GroupName,
					Kind:     "Role",
					Name:     externalSecretIntegrationRoleName,
				}, secretManagerRoleBinding.RoleRef)
				require.Equal(t, []rbacApi.Subject{{
					Kind:      rbacApi.ServiceAccountKind,
					Name:      secretIntegrationServiceAccountName,
					Namespace: stage.Spec.Namespace,
				}}, secretManagerRoleBinding.Subjects)

				requireESOObjects(t, cl, stage, esoV1)
			},
			wantErr: require.NoError,
		},
		{
			name: "eso is configured with v1beta1 when only v1beta1 is served",
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
			esoVersions: []string{"v1beta1"},
			setup: func(t *testing.T) {
				t.Setenv(platform.SecretManagerEnv, platform.SecretManagerESO)
			},
			want: func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {
				requireESOObjects(t, cl, stage, esoV1beta1)
			},
			wantErr: require.NoError,
		},
		{
			name: "eso is configured with v1 when v1 and v1beta1 are served",
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
			esoVersions: []string{"v1beta1", "v1"},
			setup: func(t *testing.T) {
				t.Setenv(platform.SecretManagerEnv, platform.SecretManagerESO)
			},
			want: func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {
				requireESOObjects(t, cl, stage, esoV1)
			},
			wantErr: require.NoError,
		},
		{
			name: "eso API is not served",
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
				t.Setenv(platform.SecretManagerEnv, platform.SecretManagerESO)
			},
			want: func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {
				err := cl.Get(context.Background(), client.ObjectKey{
					Namespace: stage.Spec.Namespace,
					Name:      secretIntegrationServiceAccountName,
				}, &corev1.ServiceAccount{})

				require.True(t, k8sErrors.IsNotFound(err))
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.ErrorIs(t, err, externalsecrets.ErrAPINotServed)
			},
		},
		{
			name: "eso: all objects already exist",
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
						Name:      secretManagerRoleBindingName("test-namespace"),
						Namespace: "default",
					},
				},
				externalsecrets.NewSecretStore(esoV1, secretStoreName, "test-namespace"),
				externalsecrets.NewExternalSecret(esoV1, externalSecretName, "test-namespace"),
			},
			esoVersions: []string{"v1"},
			setup: func(t *testing.T) {
				t.Setenv(platform.SecretManagerEnv, platform.SecretManagerESO)
			},
			want:    func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {},
			wantErr: require.NoError,
		},
		{
			name: "eso externalSecretIntegrationRole not found",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "test-namespace",
				},
			},
			objects:     []client.Object{},
			esoVersions: []string{"v1"},
			setup: func(t *testing.T) {
				t.Setenv(platform.SecretManagerEnv, platform.SecretManagerESO)
			},
			want: func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get external-secret-integration role")
			},
		},
		{
			name: "env variable SECRET_MANAGER is not set",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace:   "test-namespace",
					ClusterName: cdPipeApi.InCluster,
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
					Namespace:   "test-namespace",
					ClusterName: cdPipeApi.InCluster,
				},
			},
			objects: []client.Object{},
			setup: func(t *testing.T) {
				t.Setenv(platform.SecretManagerEnv, "invalid value")
			},
			want:    func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {},
			wantErr: require.NoError,
		},
		{
			name: "own secret manager is configured successfully",
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
				&corev1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      externalSecretName,
						Namespace: "default",
					},
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv(platform.SecretManagerEnv, platform.SecretManagerOwn)
			},
			want: func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {
				secret := &corev1.Secret{}
				err := cl.Get(context.Background(), client.ObjectKey{
					Namespace: stage.Spec.Namespace,
					Name:      externalSecretName,
				}, secret)

				require.NoError(t, err)
				require.Equal(t, map[string][]byte{"test": []byte("test")}, secret.Data)
			},
			wantErr: require.NoError,
		},
		{
			name: "own secret manager - all objects already exist",
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
				&corev1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      externalSecretName,
						Namespace: "default",
					},
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				},
				&corev1.Secret{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      externalSecretName,
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				},
			},
			setup: func(t *testing.T) {
				t.Setenv(platform.SecretManagerEnv, platform.SecretManagerOwn)
			},
			want: func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {
				secret := &corev1.Secret{}
				err := cl.Get(context.Background(), client.ObjectKey{
					Namespace: stage.Spec.Namespace,
					Name:      externalSecretName,
				}, secret)

				require.NoError(t, err)
				require.Equal(t, map[string][]byte{"test": []byte("test")}, secret.Data)
			},
			wantErr: require.NoError,
		},
		{
			name: "own secret manager - failed to get secret to copy data from",
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
				t.Setenv(platform.SecretManagerEnv, platform.SecretManagerOwn)
			},
			want: func(t *testing.T, cl client.Client, stage *cdPipeApi.Stage) {},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), fmt.Sprintf("failed to get %s secret", externalSecretName))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			sc := runtime.NewScheme()

			require.NoError(t, cdPipeApi.AddToScheme(sc))
			require.NoError(t, corev1.AddToScheme(sc))
			require.NoError(t, rbacApi.AddToScheme(sc))

			cl := fake.NewClientBuilder().
				WithScheme(sc).
				WithObjects(tt.objects...).
				WithRESTMapper(esotest.NewRESTMapper(tt.esoVersions...)).
				Build()

			h := ConfigureSecretManager{
				multiClusterClient: cl,
				internalClient:     cl,
			}

			tt.wantErr(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage))
			tt.want(t, h.multiClusterClient, tt.stage)
		})
	}
}

const (
	esoV1      = "external-secrets.io/v1"
	esoV1beta1 = "external-secrets.io/v1beta1"
)

func requireESOObjects(t *testing.T, cl client.Client, stage *cdPipeApi.Stage, apiVersion string) {
	t.Helper()

	secretStore := externalsecrets.NewSecretStore(apiVersion, secretStoreName, stage.Spec.Namespace)
	require.NoError(t, cl.Get(context.Background(), client.ObjectKeyFromObject(secretStore), secretStore))
	require.Equal(t, apiVersion, secretStore.GetAPIVersion())
	requireNestedString(t, secretStore.Object, stage.Namespace, "spec", "provider", "kubernetes", "remoteNamespace")
	requireNestedString(t, secretStore.Object, secretIntegrationServiceAccountName,
		"spec", "provider", "kubernetes", "auth", "serviceAccount", "name")

	externalSecret := externalsecrets.NewExternalSecret(apiVersion, externalSecretName, stage.Spec.Namespace)
	require.NoError(t, cl.Get(context.Background(), client.ObjectKeyFromObject(externalSecret), externalSecret))
	require.Equal(t, apiVersion, externalSecret.GetAPIVersion())
	requireNestedString(t, externalSecret.Object, secretStoreName, "spec", "secretStoreRef", "name")

	data, found, err := unstructured.NestedSlice(externalSecret.Object, "spec", "data")
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, data, 1)

	entry, ok := data[0].(map[string]interface{})
	require.True(t, ok)
	requireNestedString(t, entry, sourceSecretName, "remoteRef", "key")
}

func requireNestedString(t *testing.T, obj map[string]interface{}, want string, fields ...string) {
	t.Helper()

	got, found, err := unstructured.NestedString(obj, fields...)
	require.NoError(t, err)
	require.True(t, found, "field %v not found", fields)
	require.Equal(t, want, got, "field %v", fields)
}
