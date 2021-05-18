package chain

import (
	"context"
	"fmt"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type DeleteEnvironmentLabelFromCodebaseImageStreams struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

const lastDeletedEnvsAnnotationKey = "deploy.edp.epam.com/last-deleted-envs"

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) ServeRequest(stage *v1alpha1.Stage) error {
	log := h.log.WithValues("stage name", stage.Name)
	log.Info("start deleting environment labels from codebase image stream resources.")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd pipeline", stage.Spec.CdPipeline)
	}

	if err := h.deleteEnvironmentLabel(pipe.Spec.InputDockerStreams, pipe.Spec.Name, stage.Spec.Name, stage.Namespace); err != nil {
		return errors.Wrap(err, "couldn't set environment status")
	}

	log.Info("environment labels have been deleted from codebase image stream resources.")
	return nextServeOrNil(h.next, stage)
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) deleteEnvironmentLabel(streams []string, pipelineName, stageName, namespace string) error {
	if len(streams) == 0 {
		return fmt.Errorf("pipeline %v doesn't contain codebase image streams", pipelineName)
	}

	for _, name := range streams {
		stream, err := cluster.GetCodebaseImageStream(h.client, name, namespace)
		if err != nil {
			return errors.Wrapf(err, "couldn't get %v codebase image stream", name)
		}

		env := fmt.Sprintf("%v/%v", pipelineName, stageName)
		setAnnotation(&stream.ObjectMeta, env)
		deleteLabel(&stream.ObjectMeta, env)

		if err := h.client.Update(context.TODO(), stream); err != nil {
			return errors.Wrapf(err, "couldn't update %v codebase image stream", stream)
		}
		h.log.Info("label has been deleted from codebase image stream", "label", env, "stream", name)
	}

	return nil
}

func deleteLabel(meta *v1.ObjectMeta, label string) {
	if meta.Labels != nil {
		delete(meta.Labels, label)
	}
}

func setAnnotation(meta *v1.ObjectMeta, val string) {
	if meta.Annotations == nil {
		meta.SetAnnotations(map[string]string{})
	}
	meta.Annotations[lastDeletedEnvsAnnotationKey] = buildAnnotationValue(meta.GetAnnotations()[lastDeletedEnvsAnnotationKey], val)
}

func buildAnnotationValue(envs, val string) string {
	if len(envs) == 0 {
		return val
	}

	if findVal(strings.Split(envs, ","), val) {
		return envs
	}

	return fmt.Sprintf("%v,%v", envs, val)
}

func findVal(envs []string, val string) bool {
	for _, env := range envs {
		if env == val {
			return true
		}
	}
	return false
}
