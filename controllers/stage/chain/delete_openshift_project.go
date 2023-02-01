package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	if err := h.delete(context.TODO(), projectName); err != nil {
		return fmt.Errorf("unable to delete %v project, projectName : %w", projectName, err)
	}

	return nextServeOrNil(h.next, stage)
}

func (h DeleteOpenshiftProject) delete(ctx context.Context, name string) error {
	logger := h.log.WithValues("name", name)
	logger.Info("Trying to delete project")

	project := &projectApi.Project{}
	if err := h.client.Get(ctx, types.NamespacedName{Name: name}, project); err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("Project doesn't exist")
			return nil
		}

		return fmt.Errorf("failed to get project: %w", err)
	}

	if err := h.client.Delete(ctx, project); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	logger.Info("Project has been deleted")

	return nil
}
