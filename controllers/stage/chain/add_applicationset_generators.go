package chain

import (
	"context"
	"fmt"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

type applicationSetManager interface {
	CreateApplicationSetGenerators(ctx context.Context, stage *cdPipeApi.Stage) error
	RemoveApplicationSetGenerators(ctx context.Context, stage *cdPipeApi.Stage) error
}

type AddApplicationSetGenerators struct {
	applicationSetManager applicationSetManager
}

func (h AddApplicationSetGenerators) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	if err := h.applicationSetManager.CreateApplicationSetGenerators(ctx, stage); err != nil {
		return fmt.Errorf("failed to create application set generators: %w", err)
	}

	return nil
}
