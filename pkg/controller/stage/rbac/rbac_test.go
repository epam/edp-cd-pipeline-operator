package rbac

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	k8sApi "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	name            = "stub-name"
	namespace       = "stub-namespace"
	resourceVersion = "1"
)

func emptyRbacInit(t *testing.T) KubernetesRbac {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &k8sApi.RoleBinding{}, &k8sApi.Role{})
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	return KubernetesRbac{
		client: client,
		log:    logr.DiscardLogger{},
	}
}

func expectedRbacInit(t *testing.T) *k8sApi.RoleBinding {
	t.Helper()
	return &k8sApi.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: resourceVersion,
		},
		Subjects: nil,
		RoleRef:  k8sApi.RoleRef{},
	}
}

func expectedRoleInit(t *testing.T) *k8sApi.Role {
	t.Helper()
	return &k8sApi.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: resourceVersion,
		},
	}
}

func TestInitRbacManager_Success(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	log := ctrl.Log.WithName("rbac-manager")

	expectedRbac := KubernetesRbac{
		client: client,
		log:    log,
	}

	initializedRbac := InitRbacManager(client)
	assert.Equal(t, expectedRbac, initializedRbac)
}

func TestCreateRoleBinding_Success(t *testing.T) {
	rbac := emptyRbacInit(t)

	expectedRbac := expectedRbacInit(t)

	err := rbac.CreateRoleBinding(name, namespace, nil, k8sApi.RoleRef{})
	assert.NoError(t, err)

	createdRbac := &k8sApi.RoleBinding{}
	err = rbac.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, createdRbac)
	assert.NoError(t, err)

	assert.Equal(t, expectedRbac, createdRbac)
}

func TestGetRoleBinding_Success(t *testing.T) {
	rbac := emptyRbacInit(t)

	expectedRbac := expectedRbacInit(t)

	err := rbac.CreateRoleBinding(name, namespace, nil, k8sApi.RoleRef{})
	assert.NoError(t, err)

	createdRbac, err := rbac.GetRoleBinding(name, namespace)
	assert.NoError(t, err)

	assert.Equal(t, expectedRbac, createdRbac)
}

func TestCreateRole_Success(t *testing.T) {
	rbac := emptyRbacInit(t)

	expectedRbac := expectedRoleInit(t)

	err := rbac.CreateRole(name, namespace, nil)
	assert.NoError(t, err)

	createdRbac := &k8sApi.Role{}
	err = rbac.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, createdRbac)
	assert.NoError(t, err)

	assert.Equal(t, expectedRbac, createdRbac)
}

func TestGetRole_Success(t *testing.T) {
	rbac := emptyRbacInit(t)

	expectedRbac := expectedRoleInit(t)

	err := rbac.CreateRole(name, namespace, nil)
	assert.NoError(t, err)

	createdRbac, err := rbac.GetRole(name, namespace)
	assert.NoError(t, err)

	assert.Equal(t, expectedRbac, createdRbac)
}
