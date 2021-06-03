package chain

import (
	"context"
	"fmt"
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	edpError "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/finalizer"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutEnvironmentLabelToCodebaseImageStreams struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

func (h PutEnvironmentLabelToCodebaseImageStreams) ServeRequest(stage *cdPipeApi.Stage) error {
	log := h.log.WithValues("stage name", stage.Name)
	log.Info("start creating environment labels in codebase image stream resources.")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd pipeline", stage.Spec.CdPipeline)
	}

	if len(pipe.Spec.InputDockerStreams) == 0 {
		return fmt.Errorf("pipeline %v doesn't contain codebase image streams", pipe.Name)
	}

	for _, name := range pipe.Spec.InputDockerStreams {
		stream, err := cluster.GetCodebaseImageStream(h.client, name, stage.Namespace)
		if err != nil {
			return errors.Wrapf(err, "couldn't get %v codebase image stream", name)
		}

		if stage.IsFirst() || !finalizer.ContainsString(pipe.Spec.ApplicationsToPromote, stream.Spec.Codebase) {
			if err := h.updateLabel(stream, pipe.Name, stage.Spec.Name); err != nil {
				return err
			}
			continue
		}

		previousStage, err := util.FindPreviousStage(h.client, stage.Spec.Order, pipe.Name, stage.Namespace)
		if err != nil {
			return err
		}

		cisName := fmt.Sprintf("%v-%v-%v-verified", pipe.Name, previousStage.Spec.Name, stream.Spec.Codebase)
		verifiedStream, err := cluster.GetCodebaseImageStream(h.client, cisName, stage.Namespace)
		if err != nil {
			return edpError.CISNotFound(fmt.Sprintf("couldn't get %v codebase image stream", name))
		}

		if err := h.updateLabel(verifiedStream, pipe.Name, stage.Spec.Name); err != nil {
			return err
		}
	}

	log.Info("environment labels have been added to codebase image stream resources.")
	return nextServeOrNil(h.next, stage)
}

func (h PutEnvironmentLabelToCodebaseImageStreams) updateLabel(cis *codebaseApi.CodebaseImageStream, pipeName, stageName string) error {
	setLabel(&cis.ObjectMeta, pipeName, stageName)

	if err := h.client.Update(context.TODO(), cis); err != nil {
		return errors.Wrapf(err, "couldn't update %v codebase image stream", cis.Name)
	}

	h.log.Info("label has been added to codebase image stream",
		"label", fmt.Sprintf("%v/%v", pipeName, stageName), "stream", cis.Name)
	return nil
}

func setLabel(meta *v1.ObjectMeta, pipelineName, stageName string) {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	meta.Labels[fmt.Sprintf("%v/%v", pipelineName, stageName)] = ""
}
