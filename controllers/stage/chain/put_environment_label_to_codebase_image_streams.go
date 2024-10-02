package chain

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	edpError "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type PutEnvironmentLabelToCodebaseImageStreams struct {
	client client.Client
}

// nolint
func (h PutEnvironmentLabelToCodebaseImageStreams) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	logger := ctrl.LoggerFrom(ctx)
	if stage.IsManualTriggerType() {
		logger.Info("Trigger type is not auto deploy, skipping")
		return nil
	}

	logger.Info("start creating environment labels in codebase image stream resources.")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return fmt.Errorf("couldn't get %s cd pipeline: %w", stage.Spec.CdPipeline, err)
	}

	if len(pipe.Spec.InputDockerStreams) == 0 {
		return fmt.Errorf("pipeline %s doesn't contain codebase image streams", pipe.Name)
	}

	for _, name := range pipe.Spec.InputDockerStreams {
		stream, err := cluster.GetCodebaseImageStream(h.client, name, stage.Namespace)
		if err != nil {
			return fmt.Errorf("couldn't get %s codebase image stream: %w", name, err)
		}

		if stage.IsFirst() || !slices.Contains(pipe.Spec.ApplicationsToPromote, stream.Spec.Codebase) {
			if updErr := h.updateLabel(ctx, stream, pipe.Name, stage.Spec.Name); updErr != nil {
				return updErr
			}

			continue
		}

		if err = h.updateLabelForVerifiedStream(ctx, pipe.Name, stream.Spec.Codebase, stage); err != nil {
			return err
		}
	}

	logger.Info("environment labels have been added to codebase image stream resources.")

	return nil
}

func (h PutEnvironmentLabelToCodebaseImageStreams) updateLabelForVerifiedStream(
	ctx context.Context,
	pipeName,
	codebase string,
	stage *cdPipeApi.Stage,
) error {
	previousStageName, err := util.FindPreviousStageName(ctx, h.client, stage)
	if err != nil {
		return fmt.Errorf("failed to previous stage name: %w", err)
	}

	cisName := createCisName(pipeName, previousStageName, codebase)

	verifiedStream, err := cluster.GetCodebaseImageStream(h.client, cisName, stage.Namespace)
	if err != nil {
		return edpError.CISNotFoundError(fmt.Sprintf("failed to get %s CodebaseImageStream", cisName))
	}

	return h.updateLabel(ctx, verifiedStream, pipeName, stage.Spec.Name)
}

func (h PutEnvironmentLabelToCodebaseImageStreams) updateLabel(ctx context.Context, cis *codebaseApi.CodebaseImageStream, pipeName, stageName string) error {
	log := ctrl.LoggerFrom(ctx)

	setLabel(&cis.ObjectMeta, pipeName, stageName)

	if err := h.client.Update(ctx, cis); err != nil {
		return fmt.Errorf("couldn't update %s codebase image stream: %w", cis.Name, err)
	}

	log.Info("label has been added to codebase image stream",
		"label", fmt.Sprintf("%s/%s", pipeName, stageName), "stream", cis.Name)

	return nil
}

func setLabel(meta *metaV1.ObjectMeta, pipelineName, stageName string) {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}

	meta.Labels[createLabelName(pipelineName, stageName)] = ""
}

func createLabelName(pipeName, stageName string) string {
	return fmt.Sprintf("%s/%s", pipeName, stageName)
}

func createCisName(pipeName, previousStageName, codebase string) string {
	return fmt.Sprintf("%s-%s-%s-verified", pipeName, previousStageName, codebase)
}
