package chain

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
)

type RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

const dockerStreamsBeforeUpdateAnnotationKey = "deploy.edp.epam.com/docker-streams-before-update"

func (h RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate) ServeRequest(stage *cdPipeApi.Stage) error {
	log := h.log.WithValues("stage name", stage.Name)
	log.Info("start deleting environment labels from codebase image stream resources.")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return fmt.Errorf("failed to get %v cd pipeline: %w", stage.Spec.CdPipeline, err)
	}

	annotations := pipe.GetAnnotations()[dockerStreamsBeforeUpdateAnnotationKey]
	if annotations == "" {
		h.log.Info("CodebaseImageStream doesn't contain %v annotation." +
			" skip deleting env labels from CodebaseImageStream resources")
		return nextServeOrNil(h.next, stage)
	}

	streams := strings.Split(annotations, ",")
	for _, v := range streams {
		stream, err := cluster.GetCodebaseImageStream(h.client, v, stage.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get %v codebase image stream: %w", stream, err)
		}

		env := fmt.Sprintf("%v/%v", pipe.Name, stage.Spec.Name)
		deleteLabel(&stream.ObjectMeta, env)

		if err := h.client.Update(context.Background(), stream); err != nil {
			return fmt.Errorf("failed to update %v codebase image stream: %w", stream, err)
		}
	}

	log.Info("environment labels have been deleted from codebase image stream resources.")

	return nextServeOrNil(h.next, stage)
}
