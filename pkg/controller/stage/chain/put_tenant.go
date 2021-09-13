package chain

import (
	"context"
	"fmt"
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type PutTenant struct {
	next   handler.CdStageHandler
	space  kiosk.SpaceManager
	client client.Client
	log    logr.Logger
}

func (h PutTenant) ServeRequest(stage *cdPipeApi.Stage) error {
	name := fmt.Sprintf("%v-%v", stage.Namespace, stage.Name)
	h.log.Info("try to create namespace", "name", name)

	if err := h.createSpace(name, stage.Namespace); err != nil {
		if err := h.setFailedStatus(context.Background(), stage, err); err != nil {
			return errors.Wrapf(err, "unable to update stage %v status", stage.Name)
		}
		return errors.Wrapf(err, "unable to create %v lofk kiosk space cr", name)
	}

	return nextServeOrNil(h.next, stage)
}

func (h PutTenant) createSpace(name, account string) error {
	exists, err := h.spaceExists(name)
	if err != nil {
		return err
	}

	if exists {
		log.Info("loft kiosk space resource already exists. skip creating")
		return nil
	}

	return h.space.Create(name, account)
}

func (h PutTenant) spaceExists(name string) (bool, error) {
	h.log.Info("checking existence of space cr", "name", name)
	_, err := h.space.Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (h PutTenant) setFailedStatus(ctx context.Context, stage *cdPipeApi.Stage, err error) error {
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
		Value:           consts.FailedStatus,
	}
	return updateStatus(ctx, stage)
}
