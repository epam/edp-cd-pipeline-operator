package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

type DeleteNamespace struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

func (h DeleteNamespace) ServeRequest(stage *cdPipeApi.Stage) error {
	name := util.GenerateNamespaceName(stage)
	if err := h.delete(name); err != nil {
		return fmt.Errorf("unable to delete %v namespace, name : %w", name, err)
	}

	return nextServeOrNil(h.next, stage)
}

func (h DeleteNamespace) delete(name string) error {
	logger := h.log.WithValues("name", name)
	logger.Info("trying to delete namespace")

	ns := &v1.Namespace{}
	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Name: name,
	}, ns); err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("namespace doesn't exist")
			return nil
		}

		return fmt.Errorf("failed to get namespace: %w", err)
	}

	if err := h.client.Delete(context.TODO(), ns); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	logger.Info("namespace has been deleted")

	return nil
}
