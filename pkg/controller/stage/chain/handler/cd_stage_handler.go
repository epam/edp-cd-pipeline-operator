package handler

import (
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
)

type CdStageHandler interface {
	ServeRequest(stage *v1alpha1.Stage) error
}
