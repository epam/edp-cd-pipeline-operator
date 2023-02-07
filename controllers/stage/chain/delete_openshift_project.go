package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

// DeleteOpenshiftProject is a handler that deletes an openshift project for a stage.
type DeleteOpenshiftProject struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

// ServeRequest is a function that deletes openshift project.
func (h DeleteOpenshiftProject) ServeRequest(stage *cdPipeApi.Stage) error {
	projectName := util.GenerateNamespaceName(stage)
	logger := h.log.WithValues("name", projectName)

	project := &projectApi.Project{
		ObjectMeta: metaV1.ObjectMeta{
			Name: projectName,
		},
	}

	if err := h.client.Delete(context.TODO(), project); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Project has already been deleted")

			return nil
		}

		return fmt.Errorf("failed to delete project: %w", err)
	}

	logger.Info("Project has been deleted")

	return nextServeOrNil(h.next, stage)
}
