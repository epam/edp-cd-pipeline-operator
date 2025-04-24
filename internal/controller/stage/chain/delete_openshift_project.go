package chain

import (
	"context"
	"fmt"

	projectApi "github.com/openshift/api/project/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

// DeleteOpenshiftProject is a handler that deletes an openshift project for a stage.
type DeleteOpenshiftProject struct {
	multiClusterClient multiClusterClient
}

// ServeRequest is a function that deletes openshift project.
func (h DeleteOpenshiftProject) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	projectName := stage.Spec.Namespace
	logger := ctrl.LoggerFrom(ctx).WithValues("project", projectName)

	project := &projectApi.Project{
		ObjectMeta: metaV1.ObjectMeta{
			Name: projectName,
		},
	}

	if err := h.multiClusterClient.Delete(ctx, project); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Project has already been deleted")

			return nil
		}

		return fmt.Errorf("failed to delete project: %w", err)
	}

	logger.Info("Project has been deleted")

	return nil
}
