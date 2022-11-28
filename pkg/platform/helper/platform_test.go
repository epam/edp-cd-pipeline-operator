package helper

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPlatformTypeEnv_Success(t *testing.T) {
	stubPlatformType := "stubPlatformType"

	err := os.Setenv(platformType, stubPlatformType)
	if err != nil {
		t.Fatalf("cannot set an env variable: %v", err)
	}

	defer func() {
		err := os.Unsetenv(platformType)
		if err != nil {
			t.Fatalf("cannot unset an env variable: %v", err)
		}
	}()

	platformType := GetPlatformTypeEnv()
	assert.Equal(t, stubPlatformType, platformType)
}

func TestGetPlatformTypeEnv_PlatformTypeIsNotSet(t *testing.T) {
	err := os.Unsetenv(platformType)
	if err != nil {
		t.Fatalf("cannot unset an env variable: %v", err)
	}

	platformType := GetPlatformTypeEnv()
	assert.Equal(t, PlatformOpenshift, platformType)
}
