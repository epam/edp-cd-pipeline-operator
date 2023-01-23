package handler

import (
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

type CdStageHandler interface {
	ServeRequest(stage *cdPipeApi.Stage) error
}
