package chain

import (
	"context"
	"errors"
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
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

func TestDeleteSecretManagerRbac_ServeRequest(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, rbacApi.AddToScheme(scheme))

	stage := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: "platform-ns",
			Name:      "test-stage",
		},
		Spec: cdPipeApi.StageSpec{
			Namespace: "target-ns",
		},
	}

	roleBinding := func() *rbacApi.RoleBinding {
		return &rbacApi.RoleBinding{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "eso-target-ns",
				Namespace: "platform-ns",
			},
		}
	}

	roleBindingExists := func(t *testing.T, cl client.Client) bool {
		t.Helper()

		err := cl.Get(context.Background(), client.ObjectKeyFromObject(roleBinding()), &rbacApi.RoleBinding{})
		if k8sErrors.IsNotFound(err) {
			return false
		}

		require.NoError(t, err)

		return true
	}

	tests := []struct {
		name          string
		secretManager string
		objects       []client.Object
		funcs         interceptor.Funcs
		wantErr       assert.ErrorAssertionFunc
		wantExists    bool
	}{
		{
			name:          "role binding is deleted",
			secretManager: platform.SecretManagerESO,
			objects:       []client.Object{roleBinding()},
			wantErr:       assert.NoError,
			wantExists:    false,
		},
		{
			name:          "role binding doesn't exist",
			secretManager: platform.SecretManagerESO,
			wantErr:       assert.NoError,
			wantExists:    false,
		},
		{
			name:          "delete fails",
			secretManager: platform.SecretManagerESO,
			objects:       []client.Object{roleBinding()},
			funcs: interceptor.Funcs{
				Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
					return errors.New("connection refused")
				},
			},
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to delete eso-target-ns RoleBinding: connection refused")
			},
			wantExists: true,
		},
		{
			name:          "delete is forbidden",
			secretManager: platform.SecretManagerESO,
			objects:       []client.Object{roleBinding()},
			funcs: interceptor.Funcs{
				Delete: func(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.DeleteOption) error {
					return k8sErrors.NewForbidden(rbacApi.Resource("rolebindings"), obj.GetName(), errors.New("no delete verb"))
				},
			},
			wantErr:    assert.NoError,
			wantExists: true,
		},
		{
			name:          "secret manager is not eso",
			secretManager: platform.SecretManagerOwn,
			objects:       []client.Object{roleBinding()},
			wantErr:       assert.NoError,
			wantExists:    true,
		},
		{
			name:       "secret manager is not set",
			objects:    []client.Object{roleBinding()},
			wantErr:    assert.NoError,
			wantExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(platform.SecretManagerEnv, tt.secretManager)

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithInterceptorFuncs(tt.funcs).
				Build()

			h := DeleteSecretManagerRbac{
				multiClusterCl: cl,
			}

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantExists, roleBindingExists(t, cl))
		})
	}
}
