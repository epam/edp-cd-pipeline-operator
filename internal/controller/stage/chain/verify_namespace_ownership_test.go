package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

func TestVerifyNamespaceOwnership_ServeRequest(t *testing.T) {
	const (
		tenant   = "krci"
		targetNs = "krci-dev"
	)

	newNs := func(labels map[string]string, terminating bool) *corev1.Namespace {
		ns := &corev1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: targetNs, Labels: labels}}
		if terminating {
			ns.Status.Phase = corev1.NamespaceTerminating
		}

		return ns
	}

	tests := []struct {
		name       string
		objects    []runtime.Object
		clusterEnv string
		capsule    bool
		openshift  bool
		wantErr    require.ErrorAssertionFunc
		wantNotOwn bool
	}{
		{
			name:    "namespace absent",
			wantErr: require.NoError,
		},
		{
			name:    "owned namespace",
			objects: []runtime.Object{newNs(map[string]string{util.TenantLabelName: tenant}, false)},
			wantErr: require.NoError,
		},
		{
			name:    "terminating owned namespace",
			objects: []runtime.Object{newNs(map[string]string{util.TenantLabelName: tenant}, true)},
			wantErr: require.NoError,
		},
		{
			name:       "unlabeled namespace",
			objects:    []runtime.Object{newNs(nil, false)},
			wantErr:    require.Error,
			wantNotOwn: true,
		},
		{
			name:       "foreign tenant namespace",
			objects:    []runtime.Object{newNs(map[string]string{util.TenantLabelName: "other"}, false)},
			wantErr:    require.Error,
			wantNotOwn: true,
		},
		{
			name:       "capsule with external cluster still enforced",
			objects:    []runtime.Object{newNs(map[string]string{util.TenantLabelName: "other"}, false)},
			clusterEnv: "remote",
			capsule:    true,
			wantErr:    require.Error,
			wantNotOwn: true,
		},
		{
			name:    "capsule in-cluster foreign namespace is skipped",
			objects: []runtime.Object{newNs(map[string]string{util.TenantLabelName: "other"}, false)},
			capsule: true,
			wantErr: require.NoError,
		},
		{
			name:      "openshift is skipped",
			objects:   []runtime.Object{newNs(nil, false)},
			openshift: true,
			wantErr:   require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.capsule {
				t.Setenv(platform.TenancyEngineEnv, platform.TenancyEngineCapsule)
			}

			if tt.openshift {
				t.Setenv(platform.TypeEnv, platform.Openshift)
			}

			h := VerifyNamespaceOwnership{
				client: fake.NewClientBuilder().WithRuntimeObjects(tt.objects...).Build(),
			}

			stage := &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{Name: "dev", Namespace: tenant},
				Spec: cdPipeApi.StageSpec{
					Namespace:   targetNs,
					ClusterName: cdPipeApi.InCluster,
				},
			}
			if tt.clusterEnv != "" {
				stage.Spec.ClusterName = tt.clusterEnv
			}

			err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage)
			tt.wantErr(t, err)

			if tt.wantNotOwn {
				require.ErrorIs(t, err, util.ErrNamespaceNotOwned)
			}
		})
	}
}
