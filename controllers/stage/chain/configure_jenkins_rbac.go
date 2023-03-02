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
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
)

const (
	jenkinsAdminRbName   = "jenkins-admin"
	adminClusterRoleName = "admin"
	crNameLogKey         = "name"
)

// ConfigureJenkinsRbac configures RBAC admin roles for Jenkins.
type ConfigureJenkinsRbac struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
	rbac   rbac.Manager
}

// ServeRequest creates RoleBinding for Jenkins admin role.
func (h ConfigureJenkinsRbac) ServeRequest(stage *cdPipeApi.Stage) error {
	targetNamespace := util.GenerateNamespaceName(stage)
	logger := h.log.WithValues("stage", stage.Name, "target-ns", targetNamespace)
	logger.Info("Configuring RBAC for Jenkins")

	if err := h.rbac.CreateRoleBindingIfNotExists(
		context.TODO(),
		jenkinsAdminRbName,
		targetNamespace,
		getJenkinsAdminRoleSubjects(stage.Namespace),
		rbacApi.RoleRef{
			Name:     adminClusterRoleName,
			APIGroup: rbacApi.GroupName,
			Kind:     rbac.ClusterRoleKind,
		},
	); err != nil {
		return fmt.Errorf("failed to create %s rolebinding: %w", jenkinsAdminRbName, err)
	}

	logger.Info("RBAC for Jenkins has been configured successfully")

	return nextServeOrNil(h.next, stage)
}

func getJenkinsAdminRoleSubjects(sourceNamespace string) []rbacApi.Subject {
	const jenkinsServiceAccountName = "jenkins"

	if !platform.IsOpenshift() {
		return []rbacApi.Subject{
			{
				Kind:      rbacApi.ServiceAccountKind,
				Name:      jenkinsServiceAccountName,
				Namespace: sourceNamespace,
			},
		}
	}

	return []rbacApi.Subject{
		{
			Kind: rbacApi.GroupKind,
			Name: fmt.Sprintf("%v-edp-super-admin", sourceNamespace),
		},
		{
			Kind: rbacApi.GroupKind,
			Name: fmt.Sprintf("%v-edp-admin", sourceNamespace),
		},
		{
			Kind:      rbacApi.ServiceAccountKind,
			Name:      jenkinsServiceAccountName,
			Namespace: sourceNamespace,
		},
	}
}
