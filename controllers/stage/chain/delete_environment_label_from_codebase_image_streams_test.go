package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	edpErr "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	labelValue = "stub-data"
)

func TestServeRequest_Success(t *testing.T) {
	labels := make(map[string]string)
	labels[createLabelName(name, name)] = labelValue

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
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	deleteEnvLabel := DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline, &image).Build(),
	}

	err := deleteEnvLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.NoError(t, err)

	result, err := cluster.GetCodebaseImageStream(deleteEnvLabel.client, dockerImageName, namespace)
	assert.NoError(t, err)
	assert.Empty(t, result.Labels)
}

func TestDeleteEnvironmentLabel_VerifiedImageStream(t *testing.T) {
	stage := createStage(t, 1, cdPipeline)
	prevStage := createStage(t, 0, cdPipeline)
	prevStage.Name = previousStageName
	prevStage.Spec.Name = previousStageName
	prevStage.Labels = map[string]string{cdPipeApi.StageCdPipelineLabelName: cdPipeline}

	cisName := createCisName(name, previousStageName, codebase)

	cdPipeline := cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams:    []string{dockerImageName},
			ApplicationsToPromote: nil,
			Name:                  name,
		},
	}

	labels := make(map[string]string)
	labels[createLabelName(name, name)] = labelValue

	image := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: codebase,
		},
	}

	previousImage := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cisName,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	deleteEnvLabel := DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &prevStage, &cdPipeline, &image, &previousImage).Build(),
	}

	err := deleteEnvLabel.deleteEnvironmentLabel(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.NoError(t, err)

	previousImageStream, err := cluster.GetCodebaseImageStream(deleteEnvLabel.client, cisName, namespace)
	assert.NoError(t, err)
	assert.Empty(t, previousImageStream.Labels)

	currentImageStream, err := cluster.GetCodebaseImageStream(deleteEnvLabel.client, dockerImageName, namespace)
	assert.NoError(t, err)
	assert.Empty(t, currentImageStream.Labels)
}

func TestDeleteEnvironmentLabel_ApplicationToPromote(t *testing.T) {
	labels := make(map[string]string)
	labels[createLabelName(name, name)] = labelValue

	codebaseWithoutPromotion := "stub-codebase-non-prom"
	cisName := createCisName(name, previousStageName, codebase)

	stage := createStage(t, 1, cdPipeline)
	prevStage := createStage(t, 0, cdPipeline)
	prevStage.Name = previousStageName
	prevStage.Spec.Name = previousStageName
	prevStage.Labels = map[string]string{cdPipeApi.StageCdPipelineLabelName: cdPipeline}

	cdPipeline := cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams:    []string{dockerImageName},
			Applications:          []string{codebase, codebaseWithoutPromotion},
			ApplicationsToPromote: []string{codebase},
			Name:                  name,
		},
	}

	image := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: codebase,
		},
	}

	previousImage := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cisName,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	deleteEnvLabel := DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &prevStage, &cdPipeline, &image, &previousImage).Build(),
	}

	err := deleteEnvLabel.deleteEnvironmentLabel(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.NoError(t, err)

	previousImageStream, err := cluster.GetCodebaseImageStream(deleteEnvLabel.client, cisName, namespace)
	assert.NoError(t, err)
	assert.Empty(t, previousImageStream.Labels)

	currentImageStream, err := cluster.GetCodebaseImageStream(deleteEnvLabel.client, dockerImageName, namespace)
	assert.NoError(t, err)
	assert.Equal(t, image.Labels, currentImageStream.Labels)
}

func TestServeRequest_Error(t *testing.T) {
	stage := cdPipeApi.Stage{}

	deleteEnvLabel := DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).Build(),
	}

	err := deleteEnvLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestDeleteEnvironmentLabel_EmptyInputDockerStream(t *testing.T) {
	stage := createStage(t, 1, cdPipeline)

	cdPipeline := cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			Name: name,
		},
	}

	deleteEnvLabel := DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline).Build(),
	}

	err := deleteEnvLabel.deleteEnvironmentLabel(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.Equal(t, fmt.Errorf("pipeline %s doesn't contain codebase image streams", cdPipeline.Spec.Name), err)
}

func TestDeleteEnvironmentLabel_CantGetImageDockerStream(t *testing.T) {
	stage := createStage(t, 1, cdPipeline)

	cdPipeline := cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			Name:               name,
			InputDockerStreams: []string{dockerImageName},
		},
	}

	deleteEnvLabel := DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline).Build(),
	}

	err := deleteEnvLabel.deleteEnvironmentLabel(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestSetDeleteEnvironmentLabel_NoPreviousStageError(t *testing.T) {
	labels := make(map[string]string)
	labels[createLabelName(name, name)] = labelValue

	cisName := createCisName(name, previousStageName, codebase)

	stage := createStage(t, 1, cdPipeline)

	cdPipeline := cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams:    []string{dockerImageName},
			ApplicationsToPromote: nil,
			Name:                  name,
		},
	}

	image := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: codebase,
		},
	}

	previousImage := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cisName,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	deleteEnvLabel := DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline, &image, &previousImage).Build(),
	}

	err := deleteEnvLabel.deleteEnvironmentLabel(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.Contains(t, err.Error(), "previous stage not found")
}

func TestSetEnvLabelForVerifiedImageStream_IsNotFoundPreviousImageStream(t *testing.T) {
	stage := createStage(t, 1, cdPipeline)
	prevStage := createStage(t, 0, cdPipeline)
	prevStage.Name = previousStageName
	prevStage.Spec.Name = previousStageName
	prevStage.Labels = map[string]string{cdPipeApi.StageCdPipelineLabelName: cdPipeline}

	image := codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
		},
	}

	deleteEnvLabel := DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &prevStage, &image).Build(),
	}

	cisName := createCisName(name, previousStageName, image.Spec.Codebase)

	err := deleteEnvLabel.setEnvLabelForVerifiedImageStream(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage, &image, name, dockerImageName)
	assert.Equal(t, edpErr.CISNotFoundError(fmt.Sprintf("codebase image stream %s is not found", cisName)), err)
}
