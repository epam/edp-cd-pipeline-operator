package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	rbacApi "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
)

const (
	tenantAdminRbName = "tenant-admin"
)

type ConfigureTenantAdminRbac struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
	rbac   rbac.Manager
}

func (h ConfigureTenantAdminRbac) ServeRequest(stage *cdPipeApi.Stage) error {
	targetNamespace := util.GenerateNamespaceName(stage)
	logger := h.log.WithValues("stage", stage.Name, "target-ns", targetNamespace)
	logger.Info("Configuring tenant admin RBAC")

	if err := h.rbac.CreateRoleBindingIfNotExists(
		context.TODO(),
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

	return nextServeOrNil(h.next, stage)
}
