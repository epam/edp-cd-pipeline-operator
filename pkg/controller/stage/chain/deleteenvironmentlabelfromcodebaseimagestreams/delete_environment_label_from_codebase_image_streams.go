package deleteenvironmentlabelfromcodebaseimagestreams

import (
	"context"
	"fmt"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteEnvironmentLabelFromCodebaseImageStreams struct {
	Next   handler.CdStageHandler
	Client client.Client
}

var log = ctrl.Log.WithName("delete_environment_label_from_codebase_image_streams_chain")

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) ServeRequest(stage *v1alpha1.Stage) error {
	vLog := log.WithValues("stage name", stage.Name)
	vLog.Info("start deleting environment labels from codebase image stream resources.")

	pipe, err := util.GetCdPipeline(h.Client, stage)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd pipeline", stage.Spec.CdPipeline)
	}

	if err := h.deleteEnvironmentLabel(pipe.Spec.InputDockerStreams, pipe.Spec.Name, stage.Spec.Name, stage.Namespace); err != nil {
		return errors.Wrap(err, "couldn't set environment status")
	}

	vLog.Info("environment labels have been deleted from codebase image stream resources.")
	return nil
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) deleteEnvironmentLabel(streams []string, pipelineName, stageName, namespace string) error {
	if len(streams) == 0 {
		return fmt.Errorf("pipeline %v doesn't contain codebase image streams", pipelineName)
	}

	for _, name := range streams {
		stream, err := cluster.GetCodebaseImageStream(h.Client, name, namespace)
		if err != nil {
			return errors.Wrapf(err, "couldn't get %v codebase image stream", name)
		}

		label := fmt.Sprintf("%v/%v", pipelineName, stageName)
		deleteLabel(&stream.ObjectMeta, label)

		if err := h.Client.Update(context.TODO(), stream); err != nil {
			return errors.Wrapf(err, "couldn't update %v codebase image stream", stream)
		}
		log.Info("label has been deleted from codebase image stream", "label", label, "stream", name)
	}

	return nil
}

func deleteLabel(meta *v1.ObjectMeta, label string) {
	if meta.Labels != nil {
		delete(meta.Labels, label)
	}
}
