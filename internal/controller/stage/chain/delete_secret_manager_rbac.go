package chain

import (
	"context"
	"fmt"

	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

// DeleteSecretManagerRbac deletes the eso-<target-ns> RoleBinding created by ConfigureSecretManager.
type DeleteSecretManagerRbac struct {
	multiClusterCl multiClusterClient
}

// Runs only for SECRET_MANAGER=eso. Best effort: NotFound and Forbidden do not block Stage deletion.
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,namespace=placeholder,resources=rolebindings,verbs=delete

func (h DeleteSecretManagerRbac) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	roleBindingName := secretManagerRoleBindingName(stage.Spec.Namespace)
	logger := ctrl.LoggerFrom(ctx).WithValues("roleBindingName", roleBindingName)

	if !platform.IsSecretManagerESO() {
		logger.Info("Secret manager is not eso, skipping secret manager RoleBinding deletion")

		return nil
	}

	logger.Info("Deleting secret manager RoleBinding")

	err := h.multiClusterCl.Delete(ctx, &rbacApi.RoleBinding{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: stage.Namespace,
		},
	})

	switch {
	case err == nil:
		logger.Info("Secret manager RoleBinding has been deleted")
	case k8sErrors.IsNotFound(err):
		logger.Info("Secret manager RoleBinding has been already deleted")
	case k8sErrors.IsForbidden(err):
		logger.Error(err, "Not allowed to delete secret manager RoleBinding, leaving it in place")
	default:
		return fmt.Errorf("failed to delete %s RoleBinding: %w", roleBindingName, err)
	}

	return nil
}
