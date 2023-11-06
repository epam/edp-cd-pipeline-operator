package chain

import (
	"context"
	"fmt"

	projectApi "github.com/openshift/api/project/v1"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

// CheckNamespaceExist checks if namespace exists.
type CheckNamespaceExist struct {
	client multiClusterClient
}

// ServeRequest serves request to check if namespace/project exists.
func (h CheckNamespaceExist) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	name := stage.Spec.Namespace

	if platform.IsOpenshift() {
		if err := h.projectExist(ctx, name); err != nil {
			return err
		}
	}

	if platform.IsKubernetes() {
		if err := h.namespaceExist(ctx, name); err != nil {
			return err
		}
	}

	return nil
}

func (h CheckNamespaceExist) namespaceExist(ctx context.Context, name string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Checking existence of namespace", "name", name)

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
	log := ctrl.LoggerFrom(ctx)
	log.Info("Checking existence of project", "name", name)

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
