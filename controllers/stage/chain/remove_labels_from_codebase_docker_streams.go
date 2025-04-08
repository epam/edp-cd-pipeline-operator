package chain

import (
	"context"
	"fmt"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
)

type RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate struct {
	client client.Client
}

const dockerStreamsBeforeUpdateAnnotationKey = "deploy.edp.epam.com/docker-streams-before-update"

func (h RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	log := ctrl.LoggerFrom(ctx)
	if stage.IsManualTriggerType() {
		log.Info("Trigger type is not auto deploy, skipping")
		return nil
	}

	log.Info("start deleting environment labels from codebase image stream resources.")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return fmt.Errorf("failed to get %v cd pipeline: %w", stage.Spec.CdPipeline, err)
	}

	annotations := pipe.GetAnnotations()[dockerStreamsBeforeUpdateAnnotationKey]
	if annotations == "" {
		log.Info("CodebaseImageStream doesn't contain %v annotation." +
			" skip deleting env labels from CodebaseImageStream resources")
		return nil
	}

	streams := strings.Split(annotations, ",")
	for _, v := range streams {
		stream, err := cluster.GetCodebaseImageStreamByCodebaseBaseBranchName(ctx, h.client, v, stage.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get %v codebase image stream: %w", v, err)
		}

		env := fmt.Sprintf("%v/%v", pipe.Name, stage.Spec.Name)
		deleteLabel(&stream.ObjectMeta, env)

		if err := h.client.Update(context.Background(), stream); err != nil {
			return fmt.Errorf("failed to update %v codebase image stream: %w", stream, err)
		}
	}

	log.Info("environment labels have been deleted from codebase image stream resources.")

	return nil
}
