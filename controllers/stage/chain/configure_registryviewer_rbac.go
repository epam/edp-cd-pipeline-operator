package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	rbacApi "k8s.io/api/rbac/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
)

type ConfigureRegistryViewerRbac struct {
	next handler.CdStageHandler
	log  logr.Logger
	rbac rbac.Manager
}

func (h ConfigureRegistryViewerRbac) ServeRequest(stage *cdPipeApi.Stage) error {
	targetNamespace := stage.Spec.Namespace
	roleBindingName := generateSaRegistryViewerRoleBindingName(stage)
	logger := h.log.WithValues("stage", stage.Name, "targetNamespace", targetNamespace, "roleBindingName", roleBindingName)

	logger.Info("Configuring RoleBinding sa-registry-viewer")

	if !platform.IsOpenshift() {
		logger.Info("Skip configuring RoleBinding sa-registry-viewer for non-openshift platform")

		return nextServeOrNil(h.next, stage)
	}

	if err := h.rbac.CreateRoleBindingIfNotExists(
		context.TODO(),
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

	return nextServeOrNil(h.next, stage)
}

// generateSaRegistryViewerRoleBindingName generates name for RoleBinding for registry-viewer role.
func generateSaRegistryViewerRoleBindingName(stage *cdPipeApi.Stage) string {
	return fmt.Sprintf("%s-%s", "sa-registry-viewer", stage.Spec.Namespace)
}
