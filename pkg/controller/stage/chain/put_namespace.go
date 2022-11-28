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

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

type PutNamespace struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

func (h PutNamespace) ServeRequest(stage *cdPipeApi.Stage) error {
	name := fmt.Sprintf("%v-%v", stage.Namespace, stage.Name)
	h.log.Info("try to put namespace", crNameLogKey, name)

	if err := h.createNamespace(stage.Namespace, stage.Name); err != nil {
		if err = h.setFailedStatus(context.Background(), stage, err); err != nil {
			return fmt.Errorf("failed to update stage %v status: %w", stage.Name, err)
		}

		return fmt.Errorf("failed to create %v namespace: %w", name, err)
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

func (h PutNamespace) setFailedStatus(ctx context.Context, stage *cdPipeApi.Stage, err error) error {
	updateStatus := func(ctx context.Context, stage *cdPipeApi.Stage) error {
		if err = h.client.Status().Update(ctx, stage); err != nil {
			if err = h.client.Update(ctx, stage); err != nil {
				return fmt.Errorf("failed to update namespace status: %w", err)
			}
		}

		h.log.Info("stage status has been updated.", crNameLogKey, stage.Name)

		return nil
	}

	stage.Status = cdPipeApi.StageStatus{
		Status:          consts.FailedStatus,
		Available:       false,
		LastTimeUpdated: metaV1.Now(),
		Username:        stage.Status.Username,
		Result:          cdPipeApi.Error,
		DetailedMessage: err.Error(),
		Value:           consts.FailedStatus,
	}

	return updateStatus(ctx, stage)
}
