package chain

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/rbac"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

const (
	platformType    = "PLATFORM_TYPE"
	kubernetes      = "kubernetes"
	resourceVersion = "preCreated"
)

func createFakeClient(t *testing.T) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &k8sApi.RoleBinding{}, &k8sApi.Role{})

	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

func createConfigureRbac(t *testing.T, c client.Client, rbacManager rbac.RbacManager) ConfigureRbac {
	t.Helper()

	return ConfigureRbac{
		client: c,
		log:    logr.Discard(),
		rbac:   rbacManager,
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
	err := cdPipeApi.AddToScheme(scheme)
	require.NoError(t, err)

	err = jenkinsApi.AddToScheme(scheme)
	require.NoError(t, err)

	err = k8sApi.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&jenkinsApi.Jenkins{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "jenkins",
			Namespace: namespace,
		},
		Spec:   jenkinsApi.JenkinsSpec{},
		Status: jenkinsApi.JenkinsStatus{},
	}).Build()
	rbacManager := rbac.InitRbacManager(fakeClient)
	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	stage := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	targetNamespace := util.GenerateNamespaceName(stage)
	viewGroupRbName := generateViewGroupRbName(stage.Namespace)

	err = configureRbac.ServeRequest(stage)
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
		err = os.Unsetenv(platformType)
		if err != nil {
			t.Fatalf("unable to unset env variable: %v", err)
		}
	}()

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &k8sApi.RoleBinding{}, &k8sApi.Role{}, &cdPipeApi.Stage{})
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	stage := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	targetNamespace := util.GenerateNamespaceName(stage)
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

func TestRoleExists_True(t *testing.T) {
	fakeClient := createFakeClient(t)
	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	err := rbacManager.CreateRole(name, namespace, nil)
	assert.NoError(t, err)
	exists, err := configureRbac.roleExists(name, namespace)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestRoleExists_False(t *testing.T) {
	fakeClient := createFakeClient(t)
	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	exists, err := configureRbac.roleExists(name, namespace)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateRoleBinding_Success(t *testing.T) {
	opt := options{
		subjects: nil,
		rf:       k8sApi.RoleRef{},
	}

	fakeClient := createFakeClient(t)
	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	err := configureRbac.createRoleBinding(name, namespace, opt)
	assert.NoError(t, err)

	_, err = rbacManager.GetRoleBinding(name, namespace)
	assert.NoError(t, err)
}

func TestCreateRoleBinding_AlreadyExists(t *testing.T) {
	opt := options{
		subjects: nil,
		rf:       k8sApi.RoleRef{},
	}

	preCreatedRoleBinding := &k8sApi.RoleBinding{
		ObjectMeta: metaV1.ObjectMeta{
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

	err := configureRbac.createRoleBinding(name, namespace, opt)
	assert.Equal(t, err, nil)

	roleBinding, err := rbacManager.GetRoleBinding(name, namespace)
	assert.NoError(t, err)
	assert.Equal(t, resourceVersion, roleBinding.ResourceVersion)
}

func TestCreateRole_AlreadyExists(t *testing.T) {
	opt := options{
		subjects: nil,
		rf:       k8sApi.RoleRef{},
		pr:       []k8sApi.PolicyRule{},
	}

	preCreatedRole := &k8sApi.Role{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: resourceVersion,
		},
		Rules: []k8sApi.PolicyRule{},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &k8sApi.Role{})
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(preCreatedRole).Build()

	rbacManager := rbac.InitRbacManager(fakeClient)

	configureRbac := createConfigureRbac(t, fakeClient, rbacManager)

	err := configureRbac.createRole(name, namespace, opt)
	assert.Equal(t, err, nil)

	r, err := rbacManager.GetRole(name, namespace)
	assert.NoError(t, err)
	assert.Equal(t, resourceVersion, r.ResourceVersion)
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
			Name:     acViewRoleName,
			APIGroup: k8sApi.GroupName,
			Kind:     roleKind,
		},
		pr: []k8sApi.PolicyRule{
			{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{"apps"},
				Resources: []string{"deployments"},
			},
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
