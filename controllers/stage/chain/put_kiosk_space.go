package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

type PutKioskSpace struct {
	next   handler.CdStageHandler
	space  kiosk.SpaceManager
	client client.Client
	log    logr.Logger
}

func (h PutKioskSpace) ServeRequest(stage *cdPipeApi.Stage) error {
	name := util.GenerateNamespaceName(stage)
	h.log.Info("try to create namespace", "name", name)

	if err := h.createSpace(name, stage.Namespace); err != nil {
		if setErr := h.setFailedStatus(context.Background(), stage, err); setErr != nil {
			return fmt.Errorf("failed to update stage %s status: %w", stage.Name, err)
		}

		return fmt.Errorf("failed to create %s lofk kiosk space cr: %w", name, err)
	}

	return nextServeOrNil(h.next, stage)
}

func (h PutKioskSpace) createSpace(name, account string) error {
	exists, err := h.spaceExists(name)
	if err != nil {
		return err
	}

	if exists {
		log.Info("loft kiosk space resource already exists. skip creating")
		return nil
	}

	err = h.space.Create(name, account)
	if err != nil {
		return fmt.Errorf("failed to create kiosk space: %w", err)
	}

	return nil
}

func (h PutKioskSpace) spaceExists(name string) (bool, error) {
	h.log.Info("checking existence of space cr", "name", name)

	_, err := h.space.Get(name)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get space: %w", err)
	}

	return true, nil
}

func (h PutKioskSpace) setFailedStatus(ctx context.Context, stage *cdPipeApi.Stage, err error) error {
	updateStatus := func(ctx context.Context, stage *cdPipeApi.Stage) error {
		if err = h.client.Status().Update(ctx, stage); err != nil {
			if err = h.client.Update(ctx, stage); err != nil {
				return fmt.Errorf("failed to update kiosk space status: %w", err)
			}
		}

		h.log.Info("stage status has been updated.", "name", stage.Name)

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
