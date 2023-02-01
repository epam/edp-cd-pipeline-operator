package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

// PutOpenshiftProject is a handler that creates an openshift project for a stage.
type PutOpenshiftProject struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

// ServeRequest creates a project for a stage.
func (c PutOpenshiftProject) ServeRequest(stage *cdPipeApi.Stage) error {
	projectName := util.GenerateNamespaceName(stage)

	c.log.Info("Try to create project", crNameLogKey, projectName)

	ctx := context.TODO()
	if err := c.createProject(ctx, projectName, stage.Namespace); err != nil {
		return fmt.Errorf("failed to create %s project: %w", projectName, err)
	}

	return nextServeOrNil(c.next, stage)
}

func (c PutOpenshiftProject) createProject(ctx context.Context, name, sourceNs string) error {
	exists, err := c.projectExists(ctx, name)
	if err != nil {
		return err
	}

	if exists {
		log.Info("Project already exists, skip creating", crNameLogKey, name)
		return nil
	}

	return c.create(ctx, name, sourceNs)
}

func (c PutOpenshiftProject) projectExists(ctx context.Context, name string) (bool, error) {
	c.log.Info("Checking if project exists", crNameLogKey, name)

	if err := c.client.Get(ctx, types.NamespacedName{Name: name}, &projectApi.Project{}); err != nil {
		if k8sErrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get project: %w", err)
	}

	return true, nil
}

func (c PutOpenshiftProject) create(ctx context.Context, name, sourceNs string) error {
	logger := c.log.WithValues(crNameLogKey, name)
	logger.Info("Creating project")

	project := &projectApi.Project{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				util.TenantLabelName: sourceNs,
			},
		},
	}
	if err := c.client.Create(ctx, project); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	logger.Info("Project has been created")

	return nil
}
