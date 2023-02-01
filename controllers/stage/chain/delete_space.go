package chain

import (
	"fmt"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/kiosk"
)

type DeleteSpace struct {
	next  handler.CdStageHandler
	log   logr.Logger
	space kiosk.SpaceManager
}

func (h DeleteSpace) ServeRequest(stage *cdPipeApi.Stage) error {
	name := util.GenerateNamespaceName(stage)
	logger := h.log.WithValues("stage name", stage.Name, "space", name, "namespace", name)
	logger.Info("deleting loft kiosk space resource and namespace related to this space")

	if err := h.space.Delete(name); err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("loft kiosk space resource is already deleted")
			return nil
		}

		return fmt.Errorf("failed to delete %v loft kiosk space resource: %w", name, err)
	}

	logger.Info("namespace has been deleted.")

	return nextServeOrNil(h.next, stage)
}
