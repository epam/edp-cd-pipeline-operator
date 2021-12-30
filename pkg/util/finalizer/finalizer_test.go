package finalizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	stubStringValue    = "string"
	anotherStringValue = "anotherString"
)

func TestContainsString_Contains(t *testing.T) {
	stringSlice := []string{stubStringValue}
	containsString := ContainsString(stringSlice, stubStringValue)
	assert.True(t, containsString)
}

func TestContainsString_DoesNotContain(t *testing.T) {
	var stringSlice []string
	containsString := ContainsString(stringSlice, stubStringValue)
	assert.False(t, containsString)
}

func TestRemoveString_Success(t *testing.T) {
	stringSlice := []string{stubStringValue, anotherStringValue}
	expectedSlice := []string{anotherStringValue}
	actualSlice := RemoveString(stringSlice, stubStringValue)
	assert.Equal(t, expectedSlice, actualSlice)
}
