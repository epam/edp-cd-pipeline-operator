package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	k8sApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	edpErr "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	cdPipeline        = "stub-cdPipeline-name"
	dockerImageName   = "docker-image-name"
	previousStageName = "previous-stage"
	codebase          = "stub-codebase"
)

func createStage(t *testing.T, order int, cdPipeline string) cdPipeApi.Stage {
	t.Helper()

	return cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			Name:        name,
			Order:       order,
			CdPipeline:  cdPipeline,
			TriggerType: cdPipeApi.TriggerTypeAutoDeploy,
		},
	}
}

func schemeInit(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.Stage{}, &cdPipeApi.StageList{}, &cdPipeApi.CDPipeline{}, &codebaseApi.CodebaseImageStream{})

	return scheme
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_Success(t *testing.T) {
	stage := createStage(t, 0, cdPipeline)

	cdPipeline := cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{dockerImageName},
			Name:               name,
		},
	}

	image := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    nil,
		},
	}

	putEnvLabel := PutEnvironmentLabelToCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline, &image).Build(),
	}

	err := putEnvLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.NoError(t, err)

	imageStream, err := cluster.GetCodebaseImageStream(putEnvLabel.client, dockerImageName, namespace)
	assert.NoError(t, err)

	_, ok := imageStream.Labels[createLabelName(cdPipeline.Name, stage.Name)]
	assert.True(t, ok)
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_PreviousStageImage(t *testing.T) {
	stage := createStage(t, 1, cdPipeline)
	prevStage := createStage(t, 0, cdPipeline)
	prevStage.Name = previousStageName
	prevStage.Spec.Name = previousStageName
	prevStage.Labels = map[string]string{cdPipeApi.StageCdPipelineLabelName: cdPipeline}
	cisName := createCisName(cdPipeline, previousStageName, codebase)

	cdPipeline := cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams:    []string{dockerImageName},
			ApplicationsToPromote: []string{codebase},
			Name:                  name,
		},
	}

	image := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    nil,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: codebase,
		},
	}

	previousImage := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cisName,
			Namespace: namespace,
			Labels:    nil,
		},
	}

	putEnvLabel := PutEnvironmentLabelToCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &prevStage, &cdPipeline, &image, &previousImage).Build(),
	}

	err := putEnvLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.NoError(t, err)

	imageStream, err := cluster.GetCodebaseImageStream(putEnvLabel.client, cisName, namespace)
	assert.NoError(t, err)

	_, ok := imageStream.Labels[createLabelName(cdPipeline.Name, stage.Name)]
	assert.True(t, ok)
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_CantGetCdPipeline(t *testing.T) {
	stage := createStage(t, 0, cdPipeline)

	putEnvLabel := PutEnvironmentLabelToCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage).Build(),
	}

	err := putEnvLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_EmptyInputDockerStream(t *testing.T) {
	stage := createStage(t, 0, cdPipeline)

	cdPipeline := cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{},
			Name:               name,
		},
	}

	putEnvLabel := PutEnvironmentLabelToCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline).Build(),
	}

	err := putEnvLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.Equal(t, fmt.Errorf("pipeline %s doesn't contain codebase image streams", cdPipeline.Name), err)
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_CantGetImage(t *testing.T) {
	stage := createStage(t, 0, cdPipeline)

	cdPipeline := cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{dockerImageName},
			Name:               name,
		},
	}

	putEnvLabel := PutEnvironmentLabelToCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline).Build(),
	}

	err := putEnvLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_CantGetVerifiedStream(t *testing.T) {
	stage := createStage(t, 1, cdPipeline)
	prevStage := createStage(t, 0, cdPipeline)
	prevStage.Name = previousStageName
	prevStage.Labels = map[string]string{cdPipeApi.StageCdPipelineLabelName: cdPipeline}

	cdPipeline := cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams:    []string{dockerImageName},
			Applications:          []string{codebase},
			ApplicationsToPromote: []string{codebase},
			Name:                  name,
		},
	}

	image := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    nil,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: codebase,
		},
	}

	putEnvLabel := PutEnvironmentLabelToCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &prevStage, &cdPipeline, &image).Build(),
	}

	err := putEnvLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.Equal(t, edpErr.CISNotFoundError("failed to get stub-cdPipeline-name-stub_name-stub-codebase-verified CodebaseImageStream"), err)
}
