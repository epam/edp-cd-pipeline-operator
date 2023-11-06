package chain

import (
	"context"
	"fmt"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/kiosk"
)

type PutKioskSpace struct {
	space  kiosk.SpaceManager
	client client.Client
}

func (h PutKioskSpace) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	name := stage.Spec.Namespace
	log := ctrl.LoggerFrom(ctx).WithValues("space", name)
	log.Info("Creating kiosk space")

	if err := h.createSpace(ctrl.LoggerInto(ctx, log), name, stage.Namespace); err != nil {
		return fmt.Errorf("failed to create %s lofk kiosk space cr: %w", name, err)
	}

	return nil
}

func (h PutKioskSpace) createSpace(ctx context.Context, name, account string) error {
	log := ctrl.LoggerFrom(ctx)

	exists, err := h.spaceExists(ctx, name)
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

func (h PutKioskSpace) spaceExists(ctx context.Context, name string) (bool, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("checking existence of space cr", "name", name)

	_, err := h.space.Get(name)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get space: %w", err)
	}

	return true, nil
}
