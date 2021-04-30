package chain

import (
	"fmt"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/kiosk"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

type DeleteSpace struct {
	next  handler.CdStageHandler
	log   logr.Logger
	space kiosk.SpaceManager
}

func (h DeleteSpace) ServeRequest(stage *v1alpha1.Stage) error {
	name := fmt.Sprintf("%v-%v", stage.Namespace, stage.Name)
	log := h.log.WithValues("stage name", stage.Name, "space", name, "namespace", name)
	log.Info("deleting loft kiosk space resource and namespace related to this space")
	if err := h.space.Delete(name); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("loft kiosk space resource is already deleted")
			return nil
		}
		return errors.Wrapf(err, "unable to delete %v loft kiosk space resource", name)
	}
	log.Info("namespace has been deleted.")
	return nextServeOrNil(h.next, stage)
}
