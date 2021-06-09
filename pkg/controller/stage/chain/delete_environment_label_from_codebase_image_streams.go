package chain

import (
	"context"
	"fmt"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	edpErr "github.com/epam/edp-cd-pipeline-operator/v2/pkg/error"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/finalizer"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
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

	if err := h.deleteEnvironmentLabel(stage); err != nil {
		return errors.Wrap(err, "couldn't set environment status")
	}

	log.Info("environment labels have been deleted from codebase image stream resources.")
	return nextServeOrNil(h.next, stage)
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) deleteEnvironmentLabel(stage *v1alpha1.Stage) error {
	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd pipeline", stage.Spec.CdPipeline)
	}

	if len(pipe.Spec.InputDockerStreams) == 0 {
		return fmt.Errorf("pipeline %v doesn't contain codebase image streams", pipe.Spec.Name)
	}

	for _, name := range pipe.Spec.InputDockerStreams {
		stream, err := cluster.GetCodebaseImageStream(h.client, name, stage.Namespace)
		if err != nil {
			return errors.Wrapf(err, "couldn't get %v codebase image stream", name)
		}

		if stage.IsFirst() {
			if err := h.setEnvLabel(stage.Spec.Name, pipe.Spec.Name, stream); err != nil {
				return err
			}
			continue
		}

		if err := h.setEnvLabelForVerifiedImageStream(stage, stream, pipe.Spec.Name, name); err != nil {
			return err
		}

		if !finalizer.ContainsString(pipe.Spec.ApplicationsToPromote, stream.Spec.Codebase) {
			if err := h.setEnvLabel(stage.Spec.Name, pipe.Spec.Name, stream); err != nil {
				return err
			}
			continue
		}

	}

	return nil
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) setEnvLabel(stageName, pipeName string, stream *codebaseApi.CodebaseImageStream) error {
	env := fmt.Sprintf("%v/%v", pipeName, stageName)
	setAnnotation(&stream.ObjectMeta, env)
	deleteLabel(&stream.ObjectMeta, env)

	if err := h.client.Update(context.TODO(), stream); err != nil {
		return errors.Wrapf(err, "couldn't update %v codebase image stream", stream)
	}
	h.log.Info("label has been deleted from codebase image stream", "label", env, "stream", stream.Name)
	return nil
}

func (h DeleteEnvironmentLabelFromCodebaseImageStreams) setEnvLabelForVerifiedImageStream(stage *v1alpha1.Stage, stream *codebaseApi.CodebaseImageStream, pipeName, dockerStreamName string) error {
	previousStageName, err := util.FindPreviousStageName(stage.GetAnnotations())
	if err != nil {
		return err
	}

	cisName := fmt.Sprintf("%v-%v-%v-verified", pipeName, previousStageName, stream.Spec.Codebase)
	stream, err = cluster.GetCodebaseImageStream(h.client, cisName, stage.Namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return edpErr.CISNotFound(fmt.Sprintf("codebase image stream %v is not found", cisName))
		}
		return errors.Wrapf(err, "unable to get codebase image stream %v", stream.Name)
	}

	env := fmt.Sprintf("%v/%v", pipeName, stage.Spec.Name)
	setAnnotation(&stream.ObjectMeta, env)
	deleteLabel(&stream.ObjectMeta, env)

	if err := h.client.Update(context.TODO(), stream); err != nil {
		return errors.Wrapf(err, "couldn't update %v codebase image stream", stream)
	}

	h.log.Info("label has been deleted from codebase image stream", "label", env, "stream", dockerStreamName)
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
