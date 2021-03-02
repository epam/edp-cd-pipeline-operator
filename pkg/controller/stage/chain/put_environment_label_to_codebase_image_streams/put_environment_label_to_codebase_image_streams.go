package put_environment_label_to_codebase_image_streams

import (
	"context"
	"fmt"
	v1alphaCodebase "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	edpError "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/error"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/finalizer"
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

	if len(pipe.Spec.InputDockerStreams) == 0 {
		return fmt.Errorf("pipeline %v doesn't contain codebase image streams", pipe.Name)
	}

	for _, name := range pipe.Spec.InputDockerStreams {
		stream, err := cluster.GetCodebaseImageStream(h.Client, name, stage.Namespace)
		if err != nil {
			return errors.Wrapf(err, "couldn't get %v codebase image stream", name)
		}

		if stage.IsFirst() || !finalizer.ContainsString(pipe.Spec.ApplicationsToPromote, stream.Spec.Codebase) {
			if err := h.updateLabel(stream, pipe.Name, stage.Spec.Name); err != nil {
				return err
			}
			continue
		}

		previousStage, err := h.findPreviousStage(stage.Spec.Order, pipe.Name, stage.Namespace)
		if err != nil {
			return err
		}

		cisName := fmt.Sprintf("%v-%v-%v-verified", pipe.Name, previousStage.Spec.Name, stream.Spec.Codebase)
		verifiedStream, err := cluster.GetCodebaseImageStream(h.Client, cisName, stage.Namespace)
		if err != nil {
			return edpError.CISNotFound(fmt.Sprintf("couldn't get %v codebase image stream", name))
		}

		if err := h.updateLabel(verifiedStream, pipe.Name, stage.Spec.Name); err != nil {
			return err
		}
	}

	vLog.Info("environment labels have been added to codebase image stream resources.")
	return nil
}

func (h PutEnvironmentLabelToCodebaseImageStreams) updateLabel(cis *v1alphaCodebase.CodebaseImageStream, pipeName, stageName string) error {
	setLabel(&cis.ObjectMeta, pipeName, stageName)

	if err := h.Client.Update(context.TODO(), cis); err != nil {
		return errors.Wrapf(err, "couldn't update %v codebase image stream", cis.Name)
	}

	log.Info("label has been added to codebase image stream",
		"label", fmt.Sprintf("%v/%v", pipeName, stageName), "stream", cis.Name)
	return nil
}

func (h PutEnvironmentLabelToCodebaseImageStreams) findPreviousStage(currentOrder int, pipelineName, ns string) (*v1alpha1.Stage, error) {
	options := client.ListOptions{
		Namespace: ns,
	}
	if err := options.SetFieldSelector(fmt.Sprintf("spec.cdPipeline=%v", pipelineName)); err != nil {
		return nil, errors.Wrap(err, "couldn't set fieldSelector")
	}

	var stages v1alpha1.StageList
	if err := h.Client.List(context.TODO(), &options, &stages); err != nil {
		return nil, errors.Wrap(err, "couldn't list cd stages")
	}

	for _, s := range stages.Items {
		if s.Spec.Order == currentOrder-1 {
			log.Info("previous stage has been found", "name", s.Name)
			return &s, nil
		}
	}
	return nil, fmt.Errorf("couldn't find stage for %v cd pipeline with %v order", pipelineName, currentOrder)
}

func setLabel(meta *v1.ObjectMeta, pipelineName, stageName string) {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	meta.Labels[fmt.Sprintf("%v/%v", pipelineName, stageName)] = ""
}
