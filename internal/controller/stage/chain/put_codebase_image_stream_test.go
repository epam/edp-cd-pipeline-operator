package chain

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
)

func TestPutCodebaseImageStream_ShouldCreateCis(t *testing.T) {
	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"codebasebranch-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "CDPipeline",
				Name: "cdp-name",
			}},
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name: "stage-name",
		},
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      platform.KrciConfigMap,
			Namespace: "stub-namespace",
		},
		Data: map[string]string{
			platform.KrciConfigContainerRegistryHost:  "test-registry",
			platform.KrciConfigContainerRegistrySpace: "test-space",
		},
	}

	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				cluster.CodebaseBranchLabel: "codebasebranch-name",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdp, s, cm, cis).Build()

	cisChain := PutCodebaseImageStream{
		client: c,
	}

	err := cisChain.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), s)
	assert.NoError(t, err)

	cisResp := &codebaseApi.CodebaseImageStream{}
	err = c.Get(context.TODO(),
		types.NamespacedName{
			Name:      "cdp-name-stage-name-cb-name-verified",
			Namespace: "stub-namespace",
		},
		cisResp)
	assert.NoError(t, err)
	assert.Equal(t, "test-registry/test-space/cb-name", cisResp.Spec.ImageName)
	assert.NotNil(t, metaV1.GetControllerOf(cisResp))
}

func TestPutCodebaseImageStream_ShouldNotFindCDPipeline(t *testing.T) {
	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"cbis-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name:       "stage-name",
			CdPipeline: "non-existing-pipeline",
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdp, s).Build()

	cisChain := PutCodebaseImageStream{
		client: c,
	}

	err := cisChain.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), s)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "non-existing-pipeline") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCodebaseImageStream_ShouldNotFindRegistryUrl(t *testing.T) {
	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"cbis-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name:       "stage-name",
			CdPipeline: "cdp-name",
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdp, s).Build()

	cisChain := PutCodebaseImageStream{
		client: c,
	}

	err := cisChain.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get container registry url")
}

func TestPutCodebaseImageStream_ShouldNotFindCbis(t *testing.T) {
	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"cbis-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "CDPipeline",
				Name: "cdp-name",
			}},
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name: "stage-name",
		},
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      platform.KrciConfigMap,
			Namespace: "stub-namespace",
		},
		Data: map[string]string{
			platform.KrciConfigContainerRegistryHost:  "test-registry",
			platform.KrciConfigContainerRegistrySpace: "test-space",
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdp, s, cm).Build()

	cisChain := PutCodebaseImageStream{
		client: c,
	}

	err := cisChain.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), s)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "failed to get cbis-name codebase image stream") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCodebaseImageStream_ShouldNotFailWithExistingCbis(t *testing.T) {
	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"codebasebranch-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "CDPipeline",
				Name: "cdp-name",
			}},
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name: "stage-name",
		},
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      platform.KrciConfigMap,
			Namespace: "stub-namespace",
		},
		Data: map[string]string{
			platform.KrciConfigContainerRegistryHost:  "test-registry",
			platform.KrciConfigContainerRegistrySpace: "test-space",
		},
	}

	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				cluster.CodebaseBranchLabel: "codebasebranch-name",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	exsitingCis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name-stage-name-cb-name-verified",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdp, s, cm, cis, exsitingCis).Build()

	cisChain := PutCodebaseImageStream{
		client: c,
	}

	err := cisChain.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), s)
	assert.NoError(t, err)

	cisResp := &codebaseApi.CodebaseImageStream{}
	err = c.Get(context.TODO(),
		types.NamespacedName{
			Name:      "cdp-name-stage-name-cb-name-verified",
			Namespace: "stub-namespace",
		},
		cisResp)
	assert.NoError(t, err)
}

func TestPutCodebaseImageStream_ShouldCreateCisFromConfigMap(t *testing.T) {
	pipeline := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"codebasebranch-name",
			},
		},
	}

	stage := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "CDPipeline",
				Name: "cdp-name",
			}},
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name: "stage-name",
		},
	}

	config := &corev1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      platform.KrciConfigMap,
			Namespace: "stub-namespace",
		},
		Data: map[string]string{
			platform.KrciConfigContainerRegistryHost:  "stub-host",
			platform.KrciConfigContainerRegistrySpace: "stub-space",
		},
	}

	imageStream := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
			Labels: map[string]string{
				cluster.CodebaseBranchLabel: "codebasebranch-name",
			},
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pipeline, stage, config, imageStream).Build()

	cisChain := PutCodebaseImageStream{
		client: c,
	}

	err := cisChain.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage)
	assert.NoError(t, err)

	cisResp := &codebaseApi.CodebaseImageStream{}
	err = c.Get(
		context.Background(),
		types.NamespacedName{
			Name:      "cdp-name-stage-name-cb-name-verified",
			Namespace: "stub-namespace",
		},
		cisResp,
	)
	assert.NoError(t, err)
	assert.Equal(t, cisResp.Spec.ImageName, "stub-host/stub-space/cb-name")
}
