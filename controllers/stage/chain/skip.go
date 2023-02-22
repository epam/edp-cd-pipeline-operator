package chain

import (
	"github.com/go-logr/logr"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
)

// Skip is a stage chain element that do nothing.
type Skip struct {
	next handler.CdStageHandler
	log  logr.Logger
}

// ServeRequest does nothing.
func (c Skip) ServeRequest(stage *cdPipeApi.Stage) error {
	c.log.Info("skip chain", "name", stage.Name)

	return nextServeOrNil(c.next, stage)
}
