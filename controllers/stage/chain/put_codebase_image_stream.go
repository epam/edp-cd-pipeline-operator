package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	componentApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
)

type PutCodebaseImageStream struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

const dockerRegistryName = "docker-registry"

func (h PutCodebaseImageStream) ServeRequest(stage *cdPipeApi.Stage) error {
	logger := h.log.WithValues("stage name", stage.Name)
	logger.Info("start creating codebase image streams.")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return fmt.Errorf("failed to get %v cd pipeline: %w", stage.Spec.CdPipeline, err)
	}

	registryComponent, err := h.getDockerRegistryEdpComponent(stage.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get %v EDP component: %w", dockerRegistryName, err)
	}

	for _, ids := range pipe.Spec.InputDockerStreams {
		stream, err := cluster.GetCodebaseImageStream(h.client, ids, stage.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get %v codebase image stream: %w", ids, err)
		}

		cisName := fmt.Sprintf("%v-%v-%v-verified", pipe.Name, stage.Spec.Name, stream.Spec.Codebase)
		image := fmt.Sprintf("%v/%v/%v", registryComponent.Spec.Url, stage.Namespace, stream.Spec.Codebase)

		if err := h.createCodebaseImageStreamIfNotExists(cisName, image, stream.Spec.Codebase, stage.Namespace); err != nil {
			return fmt.Errorf("failed to create %v codebase image stream: %w", cisName, err)
		}
	}

	logger.Info("codebase image stream have been created.")

	return nextServeOrNil(h.next, stage)
}

func (h PutCodebaseImageStream) getDockerRegistryEdpComponent(namespace string) (*componentApi.EDPComponent, error) {
	ec := &componentApi.EDPComponent{}
	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Name:      dockerRegistryName,
		Namespace: namespace,
	}, ec); err != nil {
		return nil, fmt.Errorf("failed to get docker registry edp component: %w", err)
	}

	return ec, nil
}

func (h PutCodebaseImageStream) createCodebaseImageStreamIfNotExists(name, imageName, codebaseName, namespace string) error {
	cis := &codebaseApi.CodebaseImageStream{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "CodebaseImageStream",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase:  codebaseName,
			ImageName: imageName,
		},
	}

	if err := h.client.Create(context.TODO(), cis); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			h.log.Info("codebase image stream already exists. skip creating...", "name", cis.Name)
			return nil
		}

		return fmt.Errorf("failed to create codebase stream: %w", err)
	}

	h.log.Info("codebase image stream has been created", "name", name)

	return nil
}
