package chain

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sApi "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	kioskEnabledEnvVarName = "KIOSK_ENABLED"
	manualDeploy           = "manual"
	autoDeploy             = "Auto"
)

func TestChainCreation_KioskIsDisabled(t *testing.T) {
	err := os.Setenv(kioskEnabledEnvVarName, "false")
	require.NoError(t, err)

	fakeClient := createFakeClient(t)

	defManCh := CreateChain(context.Background(), fakeClient, "default", manualDeploy)
	assert.NotNil(t, defManCh)

	defAutoCh := CreateChain(context.Background(), fakeClient, "default", autoDeploy)
	assert.NotNil(t, defAutoCh)

	deleteCh := CreateDeleteChain(context.Background(), fakeClient, "default")
	assert.NotNil(t, deleteCh)
}

func TestChainCreation_KioskIsEnabled(t *testing.T) {
	err := os.Setenv(kioskEnabledEnvVarName, "true")
	require.NoError(t, err)

	fakeClient := createFakeClient(t)

	defManCh := CreateChain(context.Background(), fakeClient, "default", manualDeploy)
	assert.NotNil(t, defManCh)

	defAutoCh := CreateChain(context.Background(), fakeClient, "default", autoDeploy)
	assert.NotNil(t, defAutoCh)

	deleteCh := CreateDeleteChain(context.Background(), fakeClient, "default")
	assert.NotNil(t, deleteCh)
}

func createFakeClient(t *testing.T) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &k8sApi.RoleBinding{}, &k8sApi.Role{})

	return fake.NewClientBuilder().WithScheme(scheme).Build()
}
