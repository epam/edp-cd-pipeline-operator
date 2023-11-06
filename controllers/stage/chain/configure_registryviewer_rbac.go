package chain

import (
	"context"
	"fmt"

	rbacApi "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
)

type ConfigureRegistryViewerRbac struct {
	rbac rbac.Manager
}

func (h ConfigureRegistryViewerRbac) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	targetNamespace := stage.Spec.Namespace
	roleBindingName := generateSaRegistryViewerRoleBindingName(stage)
	logger := ctrl.LoggerFrom(ctx).WithValues("targetNamespace", targetNamespace, "roleBindingName", roleBindingName)

	logger.Info("Configuring RoleBinding sa-registry-viewer")

	if !platform.IsOpenshift() {
		logger.Info("Skip configuring RoleBinding sa-registry-viewer for non-openshift platform")

		return nil
	}

	if err := h.rbac.CreateRoleBindingIfNotExists(
		ctx,
		roleBindingName,
		stage.Namespace,
		[]rbacApi.Subject{
			{
				Kind:     rbacApi.GroupKind,
				APIGroup: rbacApi.GroupName,
				Name:     fmt.Sprintf("system:serviceaccounts:%s", targetNamespace),
			},
		},
		rbacApi.RoleRef{
			Kind:     rbac.ClusterRoleKind,
			APIGroup: rbacApi.GroupName,
			Name:     "registry-viewer",
		},
	); err != nil {
		return fmt.Errorf("failed to create %s RoleBinding: %w", roleBindingName, err)
	}

	logger.Info("RoleBinding sa-registry-viewer has been configured")

	return nil
}

// generateSaRegistryViewerRoleBindingName generates name for RoleBinding for registry-viewer role.
func generateSaRegistryViewerRoleBindingName(stage *cdPipeApi.Stage) string {
	return fmt.Sprintf("%s-%s", "sa-registry-viewer", stage.Spec.Namespace)
}
