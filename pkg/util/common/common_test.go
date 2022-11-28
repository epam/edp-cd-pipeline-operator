package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInt32P_Success(t *testing.T) {
	testInt32 := int32(0)
	int32pointer := GetInt32P(testInt32)
	assert.Equal(t, &testInt32, int32pointer)
}

func TestGetInt64P_Success(t *testing.T) {
	testInt64 := int64(0)
	int64pointer := GetInt64P(testInt64)
	assert.Equal(t, &testInt64, int64pointer)
}

func TestGetStringP_Success(t *testing.T) {
	testString := "test"
	stringPointer := GetStringP(testString)
	assert.Equal(t, &testString, stringPointer)
}
