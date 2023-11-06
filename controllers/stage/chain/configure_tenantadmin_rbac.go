package chain

import (
	"context"
	"fmt"

	rbacApi "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
)

const (
	tenantAdminRbName = "tenant-admin"
)

type ConfigureTenantAdminRbac struct {
	rbac rbac.Manager
}

func (h ConfigureTenantAdminRbac) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	targetNamespace := stage.Spec.Namespace
	logger := ctrl.LoggerFrom(ctx).WithValues("target-ns", targetNamespace)
	logger.Info("Configuring tenant admin RBAC")

	if err := h.rbac.CreateRoleBindingIfNotExists(
		ctx,
		tenantAdminRbName,
		targetNamespace,
		[]rbacApi.Subject{
			{
				APIGroup: rbacApi.GroupName,
				Kind:     rbacApi.GroupKind,
				Name:     fmt.Sprintf("%s-oidc-admins", stage.Namespace),
			},
			{
				APIGroup: rbacApi.GroupName,
				Kind:     rbacApi.GroupKind,
				Name:     fmt.Sprintf("%s-oidc-developers", stage.Namespace),
			},
		},
		rbacApi.RoleRef{
			APIGroup: rbacApi.GroupName,
			Kind:     rbac.ClusterRoleKind,
			Name:     "admin",
		},
	); err != nil {
		return fmt.Errorf("failed to create %s rolebinding: %w", tenantAdminRbName, err)
	}

	logger.Info("RBAC for tenant admin has been configured successfully")

	return nil
}
