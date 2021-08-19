package chain

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
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
	if err := os.Setenv(kioskEnabledEnvVarName, "false"); err != nil {
		panic(err)
	}

	defManCh := CreateChain(nil, manualDeploy)
	assert.NotNil(t, defManCh)

	defAutoCh := CreateChain(nil, autoDeploy)
	assert.NotNil(t, defAutoCh)

	deleteCh := CreateDeleteChain(nil)
	assert.NotNil(t, deleteCh)
}

func TestChainCreation_KioskIsEnabled(t *testing.T) {
	if err := os.Setenv(kioskEnabledEnvVarName, "true"); err != nil {
		panic(err)
	}

	defManCh := CreateChain(nil, manualDeploy)
	assert.NotNil(t, defManCh)

	defAutoCh := CreateChain(nil, autoDeploy)
	assert.NotNil(t, defAutoCh)

	deleteCh := CreateDeleteChain(nil)
	assert.NotNil(t, deleteCh)
}
