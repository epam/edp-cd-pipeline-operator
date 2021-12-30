package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ownerKind = "stub-kind"

func TestGetOwnerReference_Success(t *testing.T) {
	ownerReferences := []metav1.OwnerReference{{
		Kind: ownerKind,
	}}

	foundOwnerReference := GetOwnerReference(ownerKind, ownerReferences)
	assert.Equal(t, foundOwnerReference.Kind, ownerKind)
}

func TestGetOwnerReference_EmptySlice(t *testing.T) {
	var ownerReferences []metav1.OwnerReference

	foundOwnerReference := GetOwnerReference(ownerKind, ownerReferences)
	assert.NotEqual(t, foundOwnerReference, ownerKind)
}

func TestGetOwnerReference_IsNotFound(t *testing.T) {
	ownerReferences := []metav1.OwnerReference{{
		Kind: "uselessKind",
	}}

	foundOwnerReference := GetOwnerReference(ownerKind, ownerReferences)
	assert.NotEqual(t, foundOwnerReference, ownerKind)
}
