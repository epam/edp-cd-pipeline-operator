package chain

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	k8sApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/rbac"
)

const (
	platformType    = "PLATFORM_TYPE"
	kubernetes      = "kubernetes"
	resourceVersion = "preCreated"
)

func createFakeClient(t *testing.T) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &k8sApi.RoleBinding{})
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

func createConfigureRbac(t *testing.T, client client.Client, rbac rbac.RbacManager) ConfigureRbac {
	t.Helper()
	return ConfigureRbac{
		client: client,
		log:    logr.DiscardLogger{},
		rbac:   rbac,
	}
}

func getConfigureRbac(t *testing.T, configureRbac ConfigureRbac, name, namespace string) (*k8sApi.RoleBinding, error) {
	t.Helper()
	roleBinding := &k8sApi.RoleBinding{}
	err := configureRbac.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, roleBinding)
	return roleBinding, err
}

func TestConfigureRbac_ServeRequest_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &k8sApi.RoleBinding{}, &v1alpha1.Stage{})
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	rbacManager := rbac.InitRbacManager(fakeClient)
	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	stage := &v1alpha1.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	targetNamespace := generateTargetNamespaceName(stage)
	acViewRbName := generateAcViewRbName(targetNamespace)
	jenkinsAdminRbName := generateJenkinsAdminRbName(stage.Namespace)
	viewGroupRbName := generateViewGroupRbName(stage.Namespace)

	err := configureRbac.ServeRequest(stage)
	assert.NoError(t, err)

	_, err = getConfigureRbac(t, configureRbac, acViewRbName, targetNamespace)
	assert.NoError(t, err)

	_, err = getConfigureRbac(t, configureRbac, jenkinsAdminRbName, targetNamespace)
	assert.NoError(t, err)

	_, err = getConfigureRbac(t, configureRbac, viewGroupRbName, targetNamespace)
	assert.NoError(t, err)
}

func TestConfigureRbac_ServeRequest_DifferentPlatformType(t *testing.T) {
	err := os.Setenv(platformType, kubernetes)
	if err != nil {
		t.Fatalf("unable to set env variable: %v", err)
	}
	defer func() {
		err := os.Unsetenv(platformType)
		if err != nil {
			t.Fatalf("unable to unset env variable: %v", err)
		}
	}()

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &k8sApi.RoleBinding{}, &v1alpha1.Stage{})
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	stage := &v1alpha1.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	targetNamespace := generateTargetNamespaceName(stage)
	viewGroupRbName := generateViewGroupRbName(stage.Namespace)

	err = configureRbac.ServeRequest(stage)
	assert.NoError(t, err)

	_, err = getConfigureRbac(t, configureRbac, viewGroupRbName, targetNamespace)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestRoleBindingExists_True(t *testing.T) {
	fakeClient := createFakeClient(t)
	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	err := rbacManager.CreateRoleBinding(name, namespace, nil, k8sApi.RoleRef{})
	assert.NoError(t, err)
	exists, err := configureRbac.roleBindingExists(name, namespace)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestRoleBindingExists_False(t *testing.T) {
	fakeClient := createFakeClient(t)
	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	exists, err := configureRbac.roleBindingExists(name, namespace)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateRoleBinding_Success(t *testing.T) {
	options := options{
		subjects: nil,
		rf:       k8sApi.RoleRef{},
	}

	fakeClient := createFakeClient(t)
	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	err := configureRbac.createRoleBinding(name, namespace, options)
	assert.NoError(t, err)

	_, err = rbacManager.GetRoleBinding(name, namespace)
	assert.NoError(t, err)
}

func TestCreateRoleBinding_AlreadyExists(t *testing.T) {
	options := options{
		subjects: nil,
		rf:       k8sApi.RoleRef{},
	}

	preCreatedRoleBinding := &k8sApi.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: resourceVersion,
		},
		Subjects: nil,
		RoleRef:  k8sApi.RoleRef{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &k8sApi.RoleBinding{})
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(preCreatedRoleBinding).Build()

	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	err := configureRbac.createRoleBinding(name, namespace, options)
	assert.Equal(t, err, nil)

	roleBinding, err := rbacManager.GetRoleBinding(name, namespace)
	assert.NoError(t, err)
	assert.Equal(t, resourceVersion, roleBinding.ResourceVersion)
}

func TestBuildJenkinsAdminRoleOptions_Success(t *testing.T) {
	expectedOptions := options{
		subjects: []k8sApi.Subject{
			{
				Kind: groupKind,
				Name: fmt.Sprintf("%v-edp-super-admin", namespace),
			},
			{
				Kind: groupKind,
				Name: fmt.Sprintf("%v-edp-admin", namespace),
			},
			{
				Kind:      serviceAccountKind,
				Name:      jenkinsServiceAccountName,
				Namespace: namespace,
			},
		},
		rf: k8sApi.RoleRef{
			Name:     adminClusterRoleName,
			APIGroup: k8sApi.GroupName,
			Kind:     clusterRoleKind,
		},
	}
	actualOptions := buildJenkinsAdminRoleOptions(namespace)
	assert.Equal(t, expectedOptions, actualOptions)
}

func TestGetJenkinsAdminRoleSubjects_NotOpenshiftType(t *testing.T) {
	err := os.Setenv(platformType, kubernetes)
	if err != nil {
		t.Fatalf("unable to set env variable: %v", err)
	}
	defer func() {
		err := os.Unsetenv(platformType)
		if err != nil {
			t.Fatalf("unable to unset env variable: %v", err)
		}
	}()

	expectedSubject := []k8sApi.Subject{
		{
			Kind:      serviceAccountKind,
			Name:      jenkinsServiceAccountName,
			Namespace: namespace,
		},
	}

	actualSubject := getJenkinsAdminRoleSubjects(namespace)
	assert.Equal(t, expectedSubject, actualSubject)
}

func TestBuildAcViewRoleOptions_Success(t *testing.T) {
	expectedOptions := options{
		subjects: []k8sApi.Subject{
			{
				Kind:      serviceAccountKind,
				Name:      adminConsoleServiceAccountName,
				Namespace: namespace,
			},
		},
		rf: k8sApi.RoleRef{
			Name:     fmt.Sprintf("edp-%v-deployment-view", namespace),
			APIGroup: k8sApi.GroupName,
			Kind:     clusterRoleKind,
		},
	}

	actualOptions := buildAcViewRoleOptions(namespace)
	assert.Equal(t, expectedOptions, actualOptions)
}

func TestBuildViewGroupRoleOptions_Success(t *testing.T) {
	expectedOptions := options{
		subjects: []k8sApi.Subject{
			{
				Kind: groupKind,
				Name: fmt.Sprintf("%v-edp-view", namespace),
			},
		},
		rf: k8sApi.RoleRef{
			Name:     viewClusterRoleName,
			APIGroup: k8sApi.GroupName,
			Kind:     clusterRoleKind,
		},
	}

	actualOptions := buildViewGroupRoleOptions(namespace)
	assert.Equal(t, expectedOptions, actualOptions)
}
