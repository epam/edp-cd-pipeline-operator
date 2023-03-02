package kiosk

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NewKioskSpace creates a new unstructured.Unstructured object for a Space CRD.
func NewKioskSpace(metadata map[string]interface{}) *unstructured.Unstructured {
	space := &unstructured.Unstructured{}
	space.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "tenancy.kiosk.sh/v1alpha1",
		"metadata":   metadata,
	}

	return space
}
