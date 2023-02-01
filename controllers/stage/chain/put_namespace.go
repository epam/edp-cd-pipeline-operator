package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

type PutNamespace struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

func (h PutNamespace) ServeRequest(stage *cdPipeApi.Stage) error {
	name := util.GenerateNamespaceName(stage)
	h.log.Info("try to put namespace", crNameLogKey, name)

	if err := h.createNamespace(stage.Namespace, stage.Name); err != nil {
		return fmt.Errorf("failed to create %s namespace: %w", name, err)
	}

	return nextServeOrNil(h.next, stage)
}

func (h PutNamespace) createNamespace(sourceNs, stageName string) error {
	name := fmt.Sprintf("%v-%v", sourceNs, stageName)

	exists, err := h.namespaceExists(name)
	if err != nil {
		return err
	}

	if exists {
		log.Info("namespace already exists. skip creating", crNameLogKey, name)
		return nil
	}

	return h.create(sourceNs, stageName)
}

func (h PutNamespace) namespaceExists(name string) (bool, error) {
	h.log.Info("checking existence of namespace", crNameLogKey, name)

	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Name: name,
	}, &v1.Namespace{}); err != nil {
		if k8sErrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get namespace: %w", err)
	}

	return true, nil
}

func (h PutNamespace) create(sourceNs, stageName string) error {
	name := fmt.Sprintf("%v-%v", sourceNs, stageName)

	logger := h.log.WithValues(crNameLogKey, name)
	logger.Info("creating namespace")

	ns := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				util.TenantLabelName: sourceNs,
			},
		},
	}
	if err := h.client.Create(context.TODO(), ns); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	logger.Info("namespace is created")

	return nil
}
