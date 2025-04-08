package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func createCdPipelineWithAnnotations(t *testing.T) cdPipeApi.CDPipeline {
	t.Helper()

	annotations := make(map[string]string)
	annotations[dockerStreamsBeforeUpdateAnnotationKey] = dockerImageName

	return cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        cdPipeline,
			Namespace:   namespace,
			Annotations: annotations,
		},
	}
}

func createCodebaseImageStreamWithLabels(t *testing.T) codebaseApi.CodebaseImageStream {
	t.Helper()

	labels := make(map[string]string)
	labels[createLabelName(cdPipeline, name)] = labelValue
	labels[cluster.CodebaseImageStreamCodebaseBranchLabel] = dockerImageName

	return codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_Success(t *testing.T) {
	stage := createStage(t, 0, cdPipeline)
	cdPipe := createCdPipelineWithAnnotations(t)
	image := createCodebaseImageStreamWithLabels(t)

	removeLabel := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipe, &image).Build(),
	}

	err := removeLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.NoError(t, err)

	currentImageStream := &codebaseApi.CodebaseImageStream{}
	err = removeLabel.client.Get(
		context.Background(),
		client.ObjectKey{
			Name:      dockerImageName,
			Namespace: namespace,
		},
		currentImageStream,
	)
	require.NoError(t, err)
	assert.Empty(t, currentImageStream.Labels[createLabelName(cdPipeline, name)])
}

func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_CantGetCdPipeline(t *testing.T) {
	stage := createStage(t, 0, cdPipeline)

	removeLabel := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage).Build(),
	}

	err := removeLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_EmptyAnnotation(t *testing.T) {
	stage := createStage(t, 0, cdPipeline)

	cdPipeline := createCdPipelineWithAnnotations(t)
	cdPipeline.Annotations[dockerStreamsBeforeUpdateAnnotationKey] = ""

	image := createCodebaseImageStreamWithLabels(t)

	removeLabel := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline, &image).Build(),
	}

	err := removeLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	assert.NoError(t, err)

	currentImageStream := &codebaseApi.CodebaseImageStream{}
	err = removeLabel.client.Get(
		context.Background(),
		client.ObjectKey{
			Name:      dockerImageName,
			Namespace: namespace,
		},
		currentImageStream,
	)
	require.NoError(t, err)

	assert.NoError(t, err)
	assert.Equal(t, image.Labels, currentImageStream.Labels)
}

func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_CantGetImageStream(t *testing.T) {
	stage := createStage(t, 0, cdPipeline)

	cdPipeline := createCdPipelineWithAnnotations(t)

	removeLabel := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline).Build(),
	}

	err := removeLabel.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), &stage)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("failed to get %v codebase image stream", dockerImageName))
}
