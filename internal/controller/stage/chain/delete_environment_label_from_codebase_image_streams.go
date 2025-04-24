package chain

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
	edpErr "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
)

type DeleteEnvironmentLabelFromCodebaseImageStreams struct {
	client client.Client
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start deleting environment labels from codebase image streams")

	if err := h.deleteEnvironmentLabel(ctx, stage); err != nil {
		return fmt.Errorf("failed to set environment status: %w", err)
	}

	log.Info("Environment labels have been deleted from codebase image streams")

	return nil
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) deleteEnvironmentLabel(ctx context.Context, stage *cdPipeApi.Stage) error {
	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return fmt.Errorf("failed to get %s cd pipeline: %w", stage.Spec.CdPipeline, err)
	}

	if len(pipe.Spec.InputDockerStreams) == 0 {
		return fmt.Errorf("pipeline %s doesn't contain codebase image streams", pipe.Spec.Name)
	}

	for _, name := range pipe.Spec.InputDockerStreams {
		stream, err := cluster.GetCodebaseImageStreamByCodebaseBaseBranchName(ctx, h.client, name, stage.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get %s codebase image stream: %w", name, err)
		}

		if stage.IsFirst() {
			if envErr := h.setEnvLabel(ctx, stage.Spec.Name, pipe.Spec.Name, stream); envErr != nil {
				return envErr
			}

			continue
		}

		if envErr := h.setEnvLabelForVerifiedImageStream(ctx, stage, stream, pipe.Spec.Name, name); envErr != nil {
			return envErr
		}

		if !slices.Contains(pipe.Spec.ApplicationsToPromote, stream.Spec.Codebase) {
			if envErr := h.setEnvLabel(ctx, stage.Spec.Name, pipe.Spec.Name, stream); envErr != nil {
				return envErr
			}
		}
	}

	return nil
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) setEnvLabel(ctx context.Context, stageName, pipeName string, stream *codebaseApi.CodebaseImageStream) error {
	log := ctrl.LoggerFrom(ctx)
	env := createLabelName(pipeName, stageName)
	deleteLabel(&stream.ObjectMeta, env)

	if err := h.client.Update(ctx, stream); err != nil {
		return fmt.Errorf("failed to update %v codebase image stream: %w", stream, err)
	}

	log.Info("Label has been deleted from CodebaseImageStream", "label", env, "codebaseImageStream", stream.Name)

	return nil
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) setEnvLabelForVerifiedImageStream(ctx context.Context, stage *cdPipeApi.Stage, stream *codebaseApi.CodebaseImageStream, pipeName, dockerStreamName string) error {
	log := ctrl.LoggerFrom(ctx)

	previousStageName, err := util.FindPreviousStageName(ctx, h.client, stage)
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

	if err = h.client.Update(ctx, stream); err != nil {
		return fmt.Errorf("failed to update %v codebase image stream: %w", stream, err)
	}

	log.Info("Label has been deleted from CodebaseImageStream", "label", env, "dockerStream", dockerStreamName, "codebaseImageStream", stream.Name)

	return nil
}

func deleteLabel(meta *v1.ObjectMeta, label string) {
	if meta.Labels != nil {
		delete(meta.Labels, label)
	}
}
