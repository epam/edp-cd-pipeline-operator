package chain

import (
	"context"
	"fmt"

	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

// DeleteRegistryViewerRbac deletes sa-registry-viewer RoleBinding.
type DeleteRegistryViewerRbac struct {
	multiClusterCl multiClusterClient
}

// ServeRequest deletes sa-registry-viewer RoleBinding.
func (h DeleteRegistryViewerRbac) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	targetNamespace := stage.Spec.Namespace
	roleBindingName := generateSaRegistryViewerRoleBindingName(stage)
	logger := ctrl.LoggerFrom(ctx).WithValues("targetNamespace", targetNamespace, "roleBindingName", roleBindingName)

	logger.Info("Deleting RoleBinding sa-registry-viewer")

	if !platform.IsOpenshift() {
		logger.Info("Skip deleting RoleBinding sa-registry-viewer non-openshift platform")

		return nil
	}

	if err := h.multiClusterCl.Delete(ctx, &rbacApi.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: stage.Namespace,
		},
	}); err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("RoleBinding sa-registry-viewer has been already deleted")
			return nil
		}

		return fmt.Errorf("failed to delete %s RoleBinding: %w", roleBindingName, err)
	}

	logger.Info("RoleBinding for registry-viewer has been deleted")

	return nil
}
