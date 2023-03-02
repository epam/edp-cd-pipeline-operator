package rbac

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacApi "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateRole_Success(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, rbacApi.AddToScheme(scheme))

	type args struct {
		name      string
		namespace string
	}

	tests := []struct {
		name      string
		args      args
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		wantCheck func(t *testing.T, k8sClient client.Client)
	}{
		{
			name: "Role is created",
			args: args{
				name:      "test-role",
				namespace: "test-namespace",
			},
			wantErr: require.NoError,
			wantCheck: func(t *testing.T, k8sClient client.Client) {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Namespace: "test-namespace",
					Name:      "test-role",
				}, &rbacApi.Role{})
				require.NoError(t, err)
			},
		},
		{
			name: "Role is not created if it already exists",
			args: args{
				name:      "test-role",
				namespace: "test-namespace",
			},
			objects: []client.Object{
				&rbacApi.Role{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-namespace",
						Name:      "test-role",
					},
				},
			},
			wantErr:   require.Error,
			wantCheck: func(t *testing.T, k8sClient client.Client) {},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
			s := NewRbacManager(k8sClient, logr.Discard())

			err := s.CreateRole(
				tt.args.name,
				tt.args.namespace,
				[]rbacApi.PolicyRule{},
			)

			tt.wantErr(t, err)
			tt.wantCheck(t, k8sClient)
		})
	}
}

func TestGetRole_Success(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, rbacApi.AddToScheme(scheme))

	type args struct {
		name      string
		namespace string
	}

	tests := []struct {
		name      string
		args      args
		objects   []client.Object
		wantErr   require.ErrorAssertionFunc
		wantCheck func(t *testing.T, role *rbacApi.Role)
	}{
		{
			name: "Role exists",
			args: args{
				name:      "test-role",
				namespace: "test-namespace",
			},
			objects: []client.Object{
				&rbacApi.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-role",
						Namespace: "test-namespace",
					},
				},
			},
			wantErr: require.NoError,
			wantCheck: func(t *testing.T, role *rbacApi.Role) {
				assert.Equal(t, "test-role", role.Name)
				assert.Equal(t, "test-namespace", role.Namespace)
			},
		},
		{
			name: "Role does not exist",
			args: args{
				name:      "test-role",
				namespace: "test-namespace",
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get role")
			},
			wantCheck: func(t *testing.T, role *rbacApi.Role) {
				assert.Nil(t, role)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := NewRbacManager(
				fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build(),
				logr.Discard(),
			)

			role, err := s.GetRole(
				tt.args.name,
				tt.args.namespace,
			)
			tt.wantErr(t, err)
			tt.wantCheck(t, role)
		})
	}
}

func TestKubernetesRbac_RoleBindingExists(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, rbacApi.AddToScheme(scheme))

	tests := []struct {
		name    string
		objects []client.Object
		want    bool
	}{
		{
			name: "RoleBinding exists",
			objects: []client.Object{
				&rbacApi.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-role-binding",
						Namespace: "test-namespace",
					},
				},
			},
			want: true,
		},
		{
			name: "RoleBinding does not exist",
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := NewRbacManager(
				fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build(),
				logr.Discard(),
			)

			got, err := s.RoleBindingExists(context.Background(), "test-role-binding", "test-namespace")

			assert.Equal(t, tt.want, got)
			assert.NoError(t, err)
		})
	}
}

func TestKubernetesRbac_CreateRoleBindingIfNotExists(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, rbacApi.AddToScheme(scheme))

	type args struct {
		name      string
		namespace string
	}

	tests := []struct {
		name      string
		args      args
		objects   []client.Object
		wantErr   assert.ErrorAssertionFunc
		wantCheck func(t *testing.T, k8sClient client.Client)
	}{
		{
			name: "RoleBinding does not exist, create it",
			args: args{
				name:      "test-role-binding",
				namespace: "test-namespace",
			},
			wantErr: assert.NoError,
			wantCheck: func(t *testing.T, k8sClient client.Client) {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Namespace: "test-namespace",
					Name:      "test-role-binding",
				}, &rbacApi.RoleBinding{})
				assert.NoError(t, err)
			},
		},
		{
			name: "RoleBinding exists, do not create it",
			args: args{
				name:      "test-role-binding",
				namespace: "test-namespace",
			},
			objects: []client.Object{
				&rbacApi.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-role-binding",
						Namespace: "test-namespace",
					},
				},
			},
			wantErr: assert.NoError,
			wantCheck: func(t *testing.T, k8sClient client.Client) {
				var list rbacApi.RoleBindingList
				err := k8sClient.List(context.Background(), &list)
				assert.NoError(t, err)
				assert.Len(t, list.Items, 1)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
			s := NewRbacManager(k8sClient, logr.Discard())

			err := s.CreateRoleBindingIfNotExists(
				context.Background(),
				tt.args.name,
				tt.args.namespace,
				[]rbacApi.Subject{},
				rbacApi.RoleRef{},
			)

			tt.wantErr(t, err)
			tt.wantCheck(t, k8sClient)
		})
	}
}
