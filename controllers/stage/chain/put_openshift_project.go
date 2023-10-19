package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

// PutOpenshiftProject is a handler that creates an openshift project for a stage.
type PutOpenshiftProject struct {
	next   handler.CdStageHandler
	client multiClusterClient
	log    logr.Logger
}

// ServeRequest creates a project for a stage.
func (c PutOpenshiftProject) ServeRequest(stage *cdPipeApi.Stage) error {
	projectName := util.GenerateNamespaceName(stage)
	logger := c.log.WithValues("name", projectName)

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

			return nextServeOrNil(c.next, stage)
		}

		return fmt.Errorf("failed to create project: %w", err)
	}

	logger.Info("Project has been created")

	return nextServeOrNil(c.next, stage)
}
