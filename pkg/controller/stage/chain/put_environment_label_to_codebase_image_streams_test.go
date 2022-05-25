package chain

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	k8sApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	edpErr "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
)

const (
	cdPipeline                     = "stub-cdPipeline-name"
	dockerImageName                = "docker-image-name"
	previousStageName              = "previous Stage"
	codebase                       = "stub-codebase"
	previousStageNameAnnotationKey = "deploy.edp.epam.com/previous-stage-name"
)

func createStage(t *testing.T, order int, cdPipeline string) cdPipeApi.Stage {
	t.Helper()
	return cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			Name:       name,
			Order:      order,
			CdPipeline: cdPipeline,
		},
	}
}

func schemeInit(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.Stage{}, &cdPipeApi.CDPipeline{}, &codebaseApi.CodebaseImageStream{})
	return scheme
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_Success(t *testing.T) {
	annotations := make(map[string]string)
	annotations[previousStageNameAnnotationKey] = previousStageName

	stage := createStage(t, 0, cdPipeline)
	stage.Annotations = annotations

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
		log:    logr.DiscardLogger{},
	}

	err := putEnvLabel.ServeRequest(&stage)
	assert.NoError(t, err)

	imageStream, err := cluster.GetCodebaseImageStream(putEnvLabel.client, dockerImageName, namespace)
	_, ok := imageStream.Labels[createLabelName(cdPipeline.Name, stage.Name)]
	assert.True(t, ok)
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_PreviousStageImage(t *testing.T) {
	annotations := make(map[string]string)
	annotations[previousStageNameAnnotationKey] = previousStageName

	stage := createStage(t, 1, cdPipeline)
	stage.Annotations = annotations

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
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline, &image, &previousImage).Build(),
		log:    logr.DiscardLogger{},
	}

	err := putEnvLabel.ServeRequest(&stage)
	assert.NoError(t, err)

	imageStream, err := cluster.GetCodebaseImageStream(putEnvLabel.client, cisName, namespace)
	_, ok := imageStream.Labels[createLabelName(cdPipeline.Name, stage.Name)]
	assert.True(t, ok)
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_CantGetCdPipeline(t *testing.T) {
	stage := createStage(t, 0, cdPipeline)

	putEnvLabel := PutEnvironmentLabelToCodebaseImageStreams{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage).Build(),
		log:    logr.DiscardLogger{},
	}

	err := putEnvLabel.ServeRequest(&stage)
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
		log:    logr.DiscardLogger{},
	}

	err := putEnvLabel.ServeRequest(&stage)
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
		log:    logr.DiscardLogger{},
	}

	err := putEnvLabel.ServeRequest(&stage)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestPutEnvironmentLabelToCodebaseImageStreams_ServeRequest_CantGetPreviousStageImage(t *testing.T) {
	annotations := make(map[string]string)
	annotations[previousStageNameAnnotationKey] = previousStageName

	stage := createStage(t, 1, cdPipeline)
	stage.Annotations = annotations

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
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(&stage, &cdPipeline, &image).Build(),
		log:    logr.DiscardLogger{},
	}

	err := putEnvLabel.ServeRequest(&stage)
	assert.Equal(t, edpErr.CISNotFound(fmt.Sprintf("couldn't get %v codebase image stream", dockerImageName)), err)
}
