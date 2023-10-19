package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

// CheckNamespaceExist checks if namespace exists.
type CheckNamespaceExist struct {
	next   handler.CdStageHandler
	client multiClusterClient
	log    logr.Logger
}

// ServeRequest serves request to check if namespace/project exists.
func (h CheckNamespaceExist) ServeRequest(stage *cdPipeApi.Stage) error {
	name := util.GenerateNamespaceName(stage)

	if platform.IsOpenshift() {
		if err := h.projectExist(context.Background(), name); err != nil {
			return err
		}
	}

	if platform.IsKubernetes() {
		if err := h.namespaceExist(context.Background(), name); err != nil {
			return err
		}
	}

	return nextServeOrNil(h.next, stage)
}

func (h CheckNamespaceExist) namespaceExist(ctx context.Context, name string) error {
	h.log.Info("Checking existence of namespace", "name", name)

	if err := h.client.Get(ctx, types.NamespacedName{
		Name: name,
	}, &v1.Namespace{}); err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("namespace %s doesn't exist", name)
		}

		return fmt.Errorf("failed to check existence of %s namespace: %w", name, err)
	}

	return nil
}

func (h CheckNamespaceExist) projectExist(ctx context.Context, name string) error {
	h.log.Info("Checking existence of project", "name", name)

	if err := h.client.Get(ctx, types.NamespacedName{
		Name: name,
	}, &projectApi.Project{}); err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("project %s doesn't exist", name)
		}

		return fmt.Errorf("failed to check existence of %s project: %w", name, err)
	}

	return nil
}
