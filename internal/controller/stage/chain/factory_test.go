package chain

import (
	"context"
	"slices"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/handler"
)

func TestCreateChain(t *testing.T) {
	scheme := runtime.NewScheme()

	tests := []struct {
		name    string
		stage   *cdPipeApi.Stage
		objects []runtime.Object
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should create chain",
			stage: &cdPipeApi.Stage{
				Spec: cdPipeApi.StageSpec{
					ClusterName: cdPipeApi.InCluster,
				},
			},
			objects: []runtime.Object{},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := CreateChain(
				context.Background(),
				fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				tt.stage,
			)

			assert.NotNil(t, chain)
			tt.wantErr(t, err)
		})
	}
}

func TestCreateChain_VerifiesNamespaceOwnershipFirst(t *testing.T) {
	t.Parallel()

	ch, err := CreateChain(
		context.Background(),
		fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(),
		&cdPipeApi.Stage{
			Spec: cdPipeApi.StageSpec{
				ClusterName: cdPipeApi.InCluster,
			},
		},
	)
	require.NoError(t, err)

	c, ok := ch.(*chain)
	require.True(t, ok)
	require.NotEmpty(t, c.handlers)

	_, ok = c.handlers[0].(VerifyNamespaceOwnership)
	assert.True(t, ok, "VerifyNamespaceOwnership must be the first handler of the create chain")
}

func TestCreateDeleteChain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		stage *cdPipeApi.Stage
	}{
		{
			name: "should create delete chain",
			stage: &cdPipeApi.Stage{
				Spec: cdPipeApi.StageSpec{
					ClusterName: cdPipeApi.InCluster,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := CreateDeleteChain(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				fake.NewClientBuilder().Build(),
				&cdPipeApi.Stage{
					Spec: cdPipeApi.StageSpec{
						ClusterName: cdPipeApi.InCluster,
					},
				},
			)

			assert.NoError(t, err)
			assert.NotNil(t, chain)
		})
	}
}

func TestCreateDeleteChain_CleansUpSecretManagerRbac(t *testing.T) {
	t.Parallel()

	ch, err := CreateDeleteChain(
		ctrl.LoggerInto(context.Background(), logr.Discard()),
		fake.NewClientBuilder().Build(),
		&cdPipeApi.Stage{
			Spec: cdPipeApi.StageSpec{
				ClusterName: cdPipeApi.InCluster,
			},
		},
	)
	require.NoError(t, err)

	c, ok := ch.(*chain)
	require.True(t, ok)

	assert.True(t, slices.ContainsFunc(c.handlers, func(h handler.CdStageHandler) bool {
		_, ok := h.(DeleteSecretManagerRbac)
		return ok
	}), "DeleteSecretManagerRbac must be registered in the delete chain")
}
