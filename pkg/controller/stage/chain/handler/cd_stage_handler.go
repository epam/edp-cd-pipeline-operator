package handler

import (
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type CdStageHandler interface {
	ServeRequest(stage *v1alpha1.Stage) error
}

var log = ctrl.Log.WithName("cd_stage_handler")

func NextServeOrNil(next CdStageHandler, stage *v1alpha1.Stage) error {
	if next != nil {
		return next.ServeRequest(stage)
	}
	log.Info("handling of cd stage has been finished", "name", stage.Name)
	return nil
}
