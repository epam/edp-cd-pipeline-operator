package put_environment_label_to_codebase_image_streams

import (
	"context"
	"fmt"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type PutEnvironmentLabelToCodebaseImageStreams struct {
	Next   handler.CdStageHandler
	Client client.Client
}

var log = logf.Log.WithName("put_environment_label_to_codebase_image_streams_chain")

func (h PutEnvironmentLabelToCodebaseImageStreams) ServeRequest(stage *v1alpha1.Stage) error {
	vLog := log.WithValues("stage name", stage.Name)
	vLog.Info("start creating environment labels in codebase image stream resources.")

	pipe, err := util.GetCdPipeline(h.Client, stage)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd pipeline", stage.Spec.CdPipeline)
	}

	if err := h.setEnvironmentLabel(pipe.Spec.InputDockerStreams, pipe.Spec.Name, stage.Spec.Name, stage.Namespace); err != nil {
		return errors.Wrap(err, "couldn't set environment status")
	}

	vLog.Info("environment labels have been added to codebase image stream resources.")
	return nil
}

func (h PutEnvironmentLabelToCodebaseImageStreams) setEnvironmentLabel(streams []string, pipelineName, stageName, namespace string) error {
	if len(streams) == 0 {
		return fmt.Errorf("pipeline %v doesn't contain codebase image streams", pipelineName)
	}

	for _, name := range streams {
		stream, err := cluster.GetCodebaseImageStream(h.Client, name, namespace)
		if err != nil {
			return errors.Wrapf(err, "couldn't get %v codebase image stream", name)
		}

		setLabel(&stream.ObjectMeta, pipelineName, stageName)

		if err := h.Client.Update(context.TODO(), stream); err != nil {
			return errors.Wrapf(err, "couldn't update %v codebase image stream", stream)
		}
		log.Info("label has been added to codebase image stream",
			"label", fmt.Sprintf("%v/%v", pipelineName, stageName), "stream", name)
	}

	return nil
}

func setLabel(meta *v1.ObjectMeta, pipelineName, stageName string) {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	meta.Labels[fmt.Sprintf("%v/%v", pipelineName, stageName)] = ""
}
