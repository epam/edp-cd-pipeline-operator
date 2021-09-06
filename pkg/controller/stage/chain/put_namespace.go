package chain

import (
	"context"
	"fmt"
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type PutNamespace struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

func (h PutNamespace) ServeRequest(stage *cdPipeApi.Stage) error {
	name := fmt.Sprintf("%v-%v", stage.Namespace, stage.Name)
	h.log.Info("try to put namespace", "name", name)

	if err := h.createNamespace(stage.Namespace, stage.Name); err != nil {
		if err := h.setFailedStatus(context.Background(), stage, err); err != nil {
			return errors.Wrapf(err, "unable to update stage %v status", stage.Name)
		}
		return errors.Wrapf(err, "unable to create %v namespace", name)
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
		log.Info("namespace already exists. skip creating", "name", name)
		return nil
	}

	return h.create(sourceNs, stageName)
}

func (h PutNamespace) namespaceExists(name string) (bool, error) {
	h.log.Info("checking existence of namespace", "name", name)
	ns := &v1.Namespace{}
	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Name: name,
	}, ns); err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (h PutNamespace) create(sourceNs, stageName string) error {
	name := fmt.Sprintf("%v-%v", sourceNs, stageName)
	log := h.log.WithValues("name", name)
	log.Info("creating namespace")
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				util.TenantLabelName: sourceNs,
			},
		},
	}
	if err := h.client.Create(context.TODO(), ns); err != nil {
		return err
	}
	log.Info("namespace is created")
	return nil
}

func (h PutNamespace) setFailedStatus(ctx context.Context, stage *cdPipeApi.Stage, err error) error {
	updateStatus := func(ctx context.Context, stage *cdPipeApi.Stage) error {
		if err := h.client.Status().Update(ctx, stage); err != nil {
			if err := h.client.Update(ctx, stage); err != nil {
				return err
			}
		}
		h.log.Info("stage status has been updated.", "name", stage.Name)
		return nil
	}

	stage.Status = cdPipeApi.StageStatus{
		Status:          consts.FailedStatus,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Username:        stage.Status.Username,
		Result:          cdPipeApi.Error,
		DetailedMessage: err.Error(),
	}
	return updateStatus(ctx, stage)
}
