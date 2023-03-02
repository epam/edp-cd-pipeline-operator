package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

func TestCreateChain(t *testing.T) {
	const ns = "default"

	scheme := runtime.NewScheme()
	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "test",
		},
	}

	require.NoError(t, jenkinsApi.AddToScheme(scheme))

	tests := []struct {
		name        string
		triggerType string
		objects     []runtime.Object
	}{
		{
			name:    "should create default chain for manual deploy",
			objects: []runtime.Object{jenkins},
		},
		{
			name:        "should create default chain for auto deploy",
			triggerType: consts.AutoDeployTriggerType,
			objects:     []runtime.Object{jenkins},
		},
		{
			name: "should create tekton chain for manual deploy",
		},
		{
			name:        "should create tekton chain for auto deploy",
			triggerType: consts.AutoDeployTriggerType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := CreateChain(
				context.Background(),
				fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				ns,
				tt.triggerType,
			)

			assert.NotNil(t, chain)
		})
	}
}

func TestCreateDeleteChain(t *testing.T) {
	const ns = "default"

	scheme := runtime.NewScheme()
	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "test",
		},
	}

	require.NoError(t, jenkinsApi.AddToScheme(scheme))

	tests := []struct {
		name        string
		triggerType string
		objects     []runtime.Object
	}{
		{
			name:    "should create default chain for manual deploy",
			objects: []runtime.Object{jenkins},
		},
		{
			name:        "should create default chain for auto deploy",
			triggerType: consts.AutoDeployTriggerType,
			objects:     []runtime.Object{jenkins},
		},
		{
			name: "should create tekton chain for manual deploy",
		},
		{
			name:        "should create tekton chain for auto deploy",
			triggerType: consts.AutoDeployTriggerType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := CreateDeleteChain(
				context.Background(),
				fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				ns,
			)

			assert.NotNil(t, chain)
		})
	}
}
