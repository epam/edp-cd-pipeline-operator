package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

const (
	runtimeSecurityTenant = "krci"
	runtimeSecurityNs     = "krci-demo-dev"
)

func runtimeSecurityScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, cdPipeApi.AddToScheme(s))

	return s
}

func runtimeSecurityEnv(t *testing.T) {
	t.Helper()

	t.Setenv(platform.SecurityEnabledEnv, "true")
}

func runtimeSecurityStage(name string) *cdPipeApi.Stage {
	return &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: runtimeSecurityTenant},
		Spec:       cdPipeApi.StageSpec{Name: "dev", CdPipeline: "demo", Namespace: runtimeSecurityNs, ClusterName: cdPipeApi.InCluster},
	}
}

func targetNamespace(labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: runtimeSecurityNs, Labels: labels}}
}

func TestDeleteRuntimeSecurityRbac_ServeRequest(t *testing.T) {
	owned := map[string]string{util.TenantLabelName: runtimeSecurityTenant}
	binding := func() client.Object {
		return &rbacApi.RoleBinding{
			ObjectMeta: metaV1.ObjectMeta{Name: util.RuntimeSecurityRbName, Namespace: runtimeSecurityNs},
		}
	}

	deleting := runtimeSecurityStage("demo-qa")
	deleting.DeletionTimestamp = &metaV1.Time{Time: metaV1.Now().Time}
	deleting.Finalizers = []string{"test"}

	tests := []struct {
		name        string
		env         func(t *testing.T)
		objects     []client.Object
		wantDeleted bool
	}{
		{
			name:        "owned namespace loses the RoleBinding",
			objects:     []client.Object{targetNamespace(owned), binding()},
			wantDeleted: true,
		},
		{
			name:        "another Stage that is being deleted does not keep it",
			objects:     []client.Object{targetNamespace(owned), binding(), deleting},
			wantDeleted: true,
		},
		{
			name: "another Stage of the namespace keeps it",
			objects: []client.Object{
				targetNamespace(owned), binding(), runtimeSecurityStage("demo-qa"),
			},
		},
		{
			name:    "foreign tenant namespace keeps its RoleBinding",
			objects: []client.Object{targetNamespace(map[string]string{util.TenantLabelName: "other"}), binding()},
		},
		{
			name:        "RoleBinding already gone",
			objects:     []client.Object{targetNamespace(owned)},
			wantDeleted: true,
		},
		{
			name:        "namespace already gone",
			wantDeleted: true,
		},
		{
			name:    "disabled feature is skipped",
			env:     func(t *testing.T) { t.Setenv(platform.SecurityEnabledEnv, "false") },
			objects: []client.Object{targetNamespace(owned), binding()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeSecurityEnv(t)

			if tt.env != nil {
				tt.env(t)
			}

			stage := runtimeSecurityStage("demo-dev")
			objects := append([]client.Object{stage}, tt.objects...)
			c := fake.NewClientBuilder().WithScheme(runtimeSecurityScheme(t)).WithObjects(objects...).Build()
			h := DeleteRuntimeSecurityRbac{multiClusterClient: c, internalClient: c}

			require.NoError(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage))

			err := c.Get(context.Background(),
				client.ObjectKey{Name: util.RuntimeSecurityRbName, Namespace: runtimeSecurityNs}, &rbacApi.RoleBinding{})

			if tt.wantDeleted {
				assert.True(t, k8sErrors.IsNotFound(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteRuntimeSecurityRbac_ForbiddenDoesNotBlockDeletion(t *testing.T) {
	runtimeSecurityEnv(t)

	stage := runtimeSecurityStage("demo-dev")
	c := fake.NewClientBuilder().
		WithScheme(runtimeSecurityScheme(t)).
		WithObjects(stage, targetNamespace(map[string]string{util.TenantLabelName: runtimeSecurityTenant})).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				return k8sErrors.NewForbidden(schema.GroupResource{Resource: "rolebindings"}, util.RuntimeSecurityRbName, nil)
			},
		}).
		Build()
	h := DeleteRuntimeSecurityRbac{multiClusterClient: c, internalClient: c}

	require.NoError(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage))
}
