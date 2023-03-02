package kiosk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNewKioskSpace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metadata map[string]interface{}
		want     *unstructured.Unstructured
	}{
		{
			name: "should return a new unstructured.Unstructured object for a Space CRD",
			metadata: map[string]interface{}{
				"name": "test",
				"labels": map[string]interface{}{
					"tenant": "test",
				},
			},
			want: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Space",
					"apiVersion": "tenancy.kiosk.sh/v1alpha1",
					"metadata": map[string]interface{}{
						"name": "test",
						"labels": map[string]interface{}{
							"tenant": "test",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewKioskSpace(tt.metadata))
		})
	}
}
