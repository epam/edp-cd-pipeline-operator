package chain

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

// Skip is a stage chain element that do nothing.
type Skip struct{}

// ServeRequest does nothing.
func (Skip) ServeRequest(ctx context.Context, _ *cdPipeApi.Stage) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Skip chain")

	return nil
}
