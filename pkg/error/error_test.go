package error

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCISNotFoundError_Error(t *testing.T) {
	testStringValue := "test"
	cisNotFound := CISNotFoundError(testStringValue)

	funcResult := cisNotFound.Error()
	assert.Equal(t, testStringValue, funcResult)
}
