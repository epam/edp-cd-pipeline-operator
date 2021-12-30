package chain

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
)

type RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

const dockerStreamsBeforeUpdateAnnotationKey = "deploy.edp.epam.com/docker-streams-before-update"

func (h RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate) ServeRequest(stage *v1alpha1.Stage) error {
	log := h.log.WithValues("stage name", stage.Name)
	log.Info("start deleting environment labels from codebase image stream resources.")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd pipeline", stage.Spec.CdPipeline)
	}

	annotations := pipe.GetAnnotations()[dockerStreamsBeforeUpdateAnnotationKey]
	if len(annotations) == 0 {
		h.log.Info("CodebaseImageStream doesn't contain %v annotation." +
			" skip deleting env labels from CodebaseImageStream resources")
		return nextServeOrNil(h.next, stage)
	}

	streams := strings.Split(annotations, ",")
	for _, v := range streams {
		stream, err := cluster.GetCodebaseImageStream(h.client, v, stage.Namespace)
		if err != nil {
			return errors.Wrapf(err, "couldn't get %v codebase image stream", stream)
		}

		env := fmt.Sprintf("%v/%v", pipe.Name, stage.Spec.Name)
		deleteLabel(&stream.ObjectMeta, env)

		if err := h.client.Update(context.Background(), stream); err != nil {
			return errors.Wrapf(err, "couldn't update %v codebase image stream", stream)
		}
	}

	log.Info("environment labels have been deleted from codebase image stream resources.")
	return nextServeOrNil(h.next, stage)
}
