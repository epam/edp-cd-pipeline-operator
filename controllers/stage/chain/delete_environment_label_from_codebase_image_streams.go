package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"golang.org/x/exp/slices"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	edpErr "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type DeleteEnvironmentLabelFromCodebaseImageStreams struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) ServeRequest(stage *cdPipeApi.Stage) error {
	h.log.Info("Start deleting environment labels from codebase image streams")

	if err := h.deleteEnvironmentLabel(stage); err != nil {
		return fmt.Errorf("failed to set environment status: %w", err)
	}

	h.log.Info("Environment labels have been deleted from codebase image streams")

	return nextServeOrNil(h.next, stage)
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) deleteEnvironmentLabel(stage *cdPipeApi.Stage) error {
	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return fmt.Errorf("failed to get %s cd pipeline: %w", stage.Spec.CdPipeline, err)
	}

	if len(pipe.Spec.InputDockerStreams) == 0 {
		return fmt.Errorf("pipeline %s doesn't contain codebase image streams", pipe.Spec.Name)
	}

	for _, name := range pipe.Spec.InputDockerStreams {
		stream, err := cluster.GetCodebaseImageStream(h.client, name, stage.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get %s codebase image stream: %w", name, err)
		}

		if stage.IsFirst() {
			if envErr := h.setEnvLabel(stage.Spec.Name, pipe.Spec.Name, stream); envErr != nil {
				return envErr
			}

			continue
		}

		if envErr := h.setEnvLabelForVerifiedImageStream(stage, stream, pipe.Spec.Name, name); envErr != nil {
			return envErr
		}

		if !slices.Contains(pipe.Spec.ApplicationsToPromote, stream.Spec.Codebase) {
			if envErr := h.setEnvLabel(stage.Spec.Name, pipe.Spec.Name, stream); envErr != nil {
				return envErr
			}
		}
	}

	return nil
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) setEnvLabel(stageName, pipeName string, stream *codebaseApi.CodebaseImageStream) error {
	env := createLabelName(pipeName, stageName)
	deleteLabel(&stream.ObjectMeta, env)

	if err := h.client.Update(context.Background(), stream); err != nil {
		return fmt.Errorf("failed to update %v codebase image stream: %w", stream, err)
	}

	h.log.Info("Label has been deleted from CodebaseImageStream", "label", env, "codebaseImageStream", stream.Name)

	return nil
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) setEnvLabelForVerifiedImageStream(stage *cdPipeApi.Stage, stream *codebaseApi.CodebaseImageStream, pipeName, dockerStreamName string) error {
	previousStageName, err := util.FindPreviousStageName(context.TODO(), h.client, stage)
	if err != nil {
		return fmt.Errorf("failed to previous stage name: %w", err)
	}

	cisName := createCisName(pipeName, previousStageName, stream.Spec.Codebase)

	stream, err = cluster.GetCodebaseImageStream(h.client, cisName, stage.Namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return edpErr.CISNotFoundError(fmt.Sprintf("codebase image stream %s is not found", cisName))
		}

		return fmt.Errorf("failed to get codebase image stream %s: %w", stream.Name, err)
	}

	env := createLabelName(pipeName, stage.Spec.Name)
	deleteLabel(&stream.ObjectMeta, env)

	if err = h.client.Update(context.Background(), stream); err != nil {
		return fmt.Errorf("failed to update %v codebase image stream: %w", stream, err)
	}

	h.log.Info("Label has been deleted from CodebaseImageStream", "label", env, "dockerStream", dockerStreamName, "codebaseImageStream", stream.Name)

	return nil
}

func deleteLabel(meta *v1.ObjectMeta, label string) {
	if meta.Labels != nil {
		delete(meta.Labels, label)
	}
}
