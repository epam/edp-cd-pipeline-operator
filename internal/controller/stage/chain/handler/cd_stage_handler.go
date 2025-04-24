package handler

import (
	"context"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

type CdStageHandler interface {
	ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error
}
