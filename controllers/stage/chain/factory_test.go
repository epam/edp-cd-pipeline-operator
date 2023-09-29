package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

func TestCreateChain(t *testing.T) {
	scheme := runtime.NewScheme()

	tests := []struct {
		name    string
		stage   *cdPipeApi.Stage
		objects []runtime.Object
	}{
		{
			name: "should create default chain for manual deploy",
			stage: &cdPipeApi.Stage{
				Spec: cdPipeApi.StageSpec{
					ClusterName: cdPipeApi.InCluster,
				},
			},
			objects: []runtime.Object{},
		},
		{
			name: "should create default chain for auto deploy",
			stage: &cdPipeApi.Stage{
				Spec: cdPipeApi.StageSpec{
					TriggerType: consts.AutoDeployTriggerType,
					ClusterName: cdPipeApi.InCluster,
				},
			},
			objects: []runtime.Object{},
		},
		{
			name: "should create tekton chain for manual deploy",
			stage: &cdPipeApi.Stage{
				Spec: cdPipeApi.StageSpec{
					ClusterName: cdPipeApi.InCluster,
				},
			},
		},
		{
			name: "should create tekton chain for auto deploy",
			stage: &cdPipeApi.Stage{
				Spec: cdPipeApi.StageSpec{
					TriggerType: consts.AutoDeployTriggerType,
					ClusterName: cdPipeApi.InCluster,
				},
			},
		},
		{
			name: "should create external chain for auto deploy",
			stage: &cdPipeApi.Stage{
				Spec: cdPipeApi.StageSpec{
					TriggerType: consts.AutoDeployTriggerType,
					ClusterName: "external-cluster",
				},
			},
		},
		{
			name: "should create external chain for manual deploy",
			stage: &cdPipeApi.Stage{
				Spec: cdPipeApi.StageSpec{
					ClusterName: "external-cluster",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := CreateChain(
				context.Background(),
				fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				tt.stage,
			)

			assert.NotNil(t, chain)
		})
	}
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
		{
			name: "should create delete chain for external cluster",
			stage: &cdPipeApi.Stage{
				Spec: cdPipeApi.StageSpec{
					ClusterName: "external-cluster",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := CreateDeleteChain(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				fake.NewClientBuilder().Build(),
				tt.stage,
			)

			assert.NotNil(t, chain)
		})
	}
}
