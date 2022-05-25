package handler

import (
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
)

type CdStageHandler interface {
	ServeRequest(stage *cdPipeApi.Stage) error
}
