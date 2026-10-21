package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

func TestCheckNamespaceOwnership(t *testing.T) {
	tests := []struct {
		name    string
		labels  map[string]string
		tenant  string
		wantErr bool
	}{
		{name: "matching label", labels: map[string]string{TenantLabelName: "t"}, tenant: "t"},
		{name: "no labels", tenant: "t", wantErr: true},
		{name: "other value", labels: map[string]string{TenantLabelName: "other"}, tenant: "t", wantErr: true},
		{name: "empty value", labels: map[string]string{TenantLabelName: ""}, tenant: "t", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := &corev1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "ns-1", Labels: tt.labels}}

			err := CheckNamespaceOwnership(ns, tt.tenant)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrNamespaceNotOwned)
			assert.Contains(t, err.Error(), "namespace ns-1 exists but is not owned by tenant t: label app.edp.epam.com/tenant must be t")
		})
	}
}

func TestOwnershipEnforced(t *testing.T) {
	tests := []struct {
		name     string
		engine   string
		platform string
		cluster  string
		want     bool
	}{
		{name: "capsule in-cluster", engine: "capsule", cluster: "in-cluster", want: false},
		{name: "capsule empty cluster", engine: "capsule", cluster: "", want: false},
		{name: "capsule remote", engine: "capsule", cluster: "remote", want: true},
		{name: "none in-cluster", engine: "none", cluster: "in-cluster", want: true},
		{name: "openshift in-cluster", engine: "none", platform: "openshift", cluster: "in-cluster", want: false},
		{name: "openshift remote", engine: "none", platform: "openshift", cluster: "remote", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TENANCY_ENGINE", tt.engine)
			t.Setenv("PLATFORM_TYPE", tt.platform)

			stage := &cdPipeApi.Stage{Spec: cdPipeApi.StageSpec{ClusterName: tt.cluster}}

			assert.Equal(t, tt.want, OwnershipEnforced(stage))
		})
	}
}
