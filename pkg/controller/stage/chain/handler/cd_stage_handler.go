package handler

import (
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type CdStageHandler interface {
	ServeRequest(stage *v1alpha1.Stage) error
}

var log = logf.Log.WithName("cd_stage_handler")

func NextServeOrNil(next CdStageHandler, stage *v1alpha1.Stage) error {
	if next != nil {
		return next.ServeRequest(stage)
	}
	log.Info("handling of cd stage has been finished", "name", stage.Name)
	return nil
}
