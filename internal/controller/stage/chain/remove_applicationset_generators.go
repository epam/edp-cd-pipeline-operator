package chain

import (
	"context"
	"fmt"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

type RemoveApplicationSetGenerators struct {
	applicationSetManager applicationSetManager
}

func (h RemoveApplicationSetGenerators) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	if err := h.applicationSetManager.RemoveApplicationSetGenerators(ctx, stage); err != nil {
		return fmt.Errorf("failed to remove application set generators: %w", err)
	}

	return nil
}
