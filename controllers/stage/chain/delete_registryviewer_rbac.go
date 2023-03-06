package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

// DeleteRegistryViewerRbac deletes sa-registry-viewer RoleBinding.
type DeleteRegistryViewerRbac struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

// ServeRequest deletes sa-registry-viewer RoleBinding.
func (h DeleteRegistryViewerRbac) ServeRequest(stage *cdPipeApi.Stage) error {
	targetNamespace := util.GenerateNamespaceName(stage)
	roleBindingName := generateSaRegistryViewerRoleBindingName(stage)
	logger := h.log.WithValues("stage", stage.Name, "targetNamespace", targetNamespace, "roleBindingName", roleBindingName)

	logger.Info("Deleting RoleBinding sa-registry-viewer")

	if !platform.IsOpenshift() {
		logger.Info("Skip deleting RoleBinding sa-registry-viewer non-openshift platform")

		return nextServeOrNil(h.next, stage)
	}

	if err := h.client.Delete(context.TODO(), &rbacApi.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: stage.Namespace,
		},
	}); err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("RoleBinding sa-registry-viewer has been already deleted")
			return nextServeOrNil(h.next, stage)
		}

		return fmt.Errorf("failed to delete %s RoleBinding: %w", roleBindingName, err)
	}

	logger.Info("RoleBinding for registry-viewer has been deleted")

	return nextServeOrNil(h.next, stage)
}
