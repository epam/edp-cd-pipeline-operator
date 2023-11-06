package chain

import (
	"context"
	"fmt"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/kiosk"
)

type DeleteSpace struct {
	space kiosk.SpaceManager
}

func (h DeleteSpace) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	name := stage.Spec.Namespace
	logger := ctrl.LoggerFrom(ctx).WithValues("space", name, "namespace", name)
	logger.Info("deleting loft kiosk space resource and namespace related to this space")

	if err := h.space.Delete(name); err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("loft kiosk space resource is already deleted")
			return nil
		}

		return fmt.Errorf("failed to delete %v loft kiosk space resource: %w", name, err)
	}

	logger.Info("namespace has been deleted.")

	return nil
}
