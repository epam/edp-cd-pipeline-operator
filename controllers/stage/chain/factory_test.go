package chain

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	kioskEnabledEnvVarName = "KIOSK_ENABLED"
	manualDeploy           = "manual"
	autoDeploy             = "Auto"
)

func TestKioskEnabled_VarIsNotSet(t *testing.T) {
	assert.False(t, kioskEnabled())
}

func TestKioskEnabled_VarIsFalse(t *testing.T) {
	if err := os.Setenv(kioskEnabledEnvVarName, "false"); err != nil {
		panic(err)
	}

	assert.False(t, kioskEnabled())
}

func TestKioskEnabled_VarIsTrue(t *testing.T) {
	if err := os.Setenv(kioskEnabledEnvVarName, "true"); err != nil {
		panic(err)
	}

	assert.True(t, kioskEnabled())
}

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
