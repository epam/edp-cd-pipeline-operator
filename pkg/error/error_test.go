package error

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCISNotFound_Error(t *testing.T) {
	testStringValue := "test"
	cisNotFound := CISNotFound(testStringValue)

	funcResult := cisNotFound.Error()
	assert.Equal(t, testStringValue, funcResult)
}
