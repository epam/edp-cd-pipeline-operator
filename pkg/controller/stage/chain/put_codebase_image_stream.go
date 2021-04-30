package chain

import (
	"context"
	"fmt"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	v1alphaEdpComponent "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PutCodebaseImageStream struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

const dockerRegistryName = "docker-registry"

func (h PutCodebaseImageStream) ServeRequest(stage *v1alpha1.Stage) error {
	log := h.log.WithValues("stage name", stage.Name)
	log.Info("start creating codebase image streams.")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v cd pipeline", stage.Spec.CdPipeline)
	}

	registryComponent, err := h.getDockerRegistryEdpComponent(stage.Namespace)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %v EDP component", dockerRegistryName)
	}

	for _, name := range pipe.Spec.ApplicationsToPromote {
		cisName := fmt.Sprintf("%v-%v-%v-verified", pipe.Name, stage.Spec.Name, name)
		image := fmt.Sprintf("%v/%v", registryComponent.Spec.Url, stage.Namespace)
		if err := h.createCodebaseImageStreamIfNotExists(cisName, image, name, stage.Namespace); err != nil {
			return errors.Wrapf(err, "couldn't create %v codebase image stream", cisName)
		}
	}
	log.Info("codebase image stream have been created.")
	return nextServeOrNil(h.next, stage)
}

func (h PutCodebaseImageStream) getDockerRegistryEdpComponent(namespace string) (*v1alphaEdpComponent.EDPComponent, error) {
	ec := &v1alphaEdpComponent.EDPComponent{}
	err := h.client.Get(context.TODO(), types.NamespacedName{
		Name:      dockerRegistryName,
		Namespace: namespace,
	}, ec)
	if err != nil {
		return nil, err
	}
	return ec, nil
}

func (h PutCodebaseImageStream) createCodebaseImageStreamIfNotExists(name, imageName, codebaseName, namespace string) error {
	cis := &codebaseApi.CodebaseImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "CodebaseImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase:  codebaseName,
			ImageName: imageName,
		},
	}

	if err := h.client.Create(context.TODO(), cis); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			h.log.Info("codebase image stream already exists. skip creating...", "name", cis.Name)
			return nil
		}
		return err
	}
	h.log.Info("codebase image stream has been created", "name", name)
	return nil
}
