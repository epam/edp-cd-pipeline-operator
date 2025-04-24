package chain

import (
	"context"
	"fmt"

	projectApi "github.com/openshift/api/project/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
)

// PutOpenshiftProject is a handler that creates an openshift project for a stage.
type PutOpenshiftProject struct {
	client multiClusterClient
}

// ServeRequest creates a project for a stage.
func (c PutOpenshiftProject) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	projectName := stage.Spec.Namespace
	logger := ctrl.LoggerFrom(ctx).WithValues("project", projectName)

	logger.Info("Try to create project")

	project := &projectApi.ProjectRequest{
		ObjectMeta: metaV1.ObjectMeta{
			Name: projectName,
			Labels: map[string]string{
				util.TenantLabelName: stage.Namespace,
			},
		},
	}

	if err := c.client.Create(context.TODO(), project); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("Project already exists")

			return nil
		}

		return fmt.Errorf("failed to create project: %w", err)
	}

	logger.Info("Project has been created")

	return nil
}
