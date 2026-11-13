package chain

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
)

// DeleteRuntimeSecurityRbac removes the runtime security RoleBinding from a Stage namespace that outlives the Stage.
type DeleteRuntimeSecurityRbac struct {
	multiClusterClient multiClusterClient
	internalClient     client.Client
}

// ServeRequest deletes the RoleBinding unless the namespace is not owned by the tenant
// or another Stage of the tenant still uses it.
func (h DeleteRuntimeSecurityRbac) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	targetNamespace := stage.Spec.Namespace
	logger := ctrl.LoggerFrom(ctx).WithValues("target-ns", targetNamespace)

	if !util.RuntimeSecurityApplies(stage) {
		logger.Info("Skip deleting runtime security RBAC")

		return nil
	}

	if util.OwnershipEnforced(stage) {
		ns := &corev1.Namespace{}
		if err := h.multiClusterClient.Get(ctx, client.ObjectKey{Name: targetNamespace}, ns); err != nil {
			return client.IgnoreNotFound(fmt.Errorf("failed to get namespace %s: %w", targetNamespace, err))
		}

		if err := util.CheckNamespaceOwnership(ns, stage.Namespace); err != nil {
			logger.Info("Namespace is not owned by the tenant, skipping deletion of runtime security RBAC",
				"reason", err.Error())

			return nil
		}
	}

	inUse, err := h.namespaceInUse(ctx, stage)
	if err != nil {
		return err
	}

	if inUse {
		logger.Info("Namespace is used by another Stage, keeping runtime security RBAC")

		return nil
	}

	err = h.multiClusterClient.Delete(ctx, &rbacApi.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: util.RuntimeSecurityRbName, Namespace: targetNamespace},
	})

	switch {
	case err == nil:
		logger.Info("Runtime security RBAC has been deleted")
	case k8sErrors.IsNotFound(err):
		logger.Info("Runtime security RBAC has been already deleted")
	case k8sErrors.IsForbidden(err):
		// The reader RoleBinding only grants reads of a namespace that is going away, so it must not block Stage deletion.
		logger.Error(err, "Not allowed to delete runtime security RoleBinding, leaving it in place")
	default:
		return fmt.Errorf("failed to delete %s rolebinding: %w", util.RuntimeSecurityRbName, err)
	}

	return nil
}

// namespaceInUse reports whether another Stage of the tenant that is not being deleted targets the same namespace.
func (h DeleteRuntimeSecurityRbac) namespaceInUse(ctx context.Context, stage *cdPipeApi.Stage) (bool, error) {
	stages := &cdPipeApi.StageList{}
	if err := h.internalClient.List(ctx, stages, client.InNamespace(stage.Namespace)); err != nil {
		return false, fmt.Errorf("failed to list stages: %w", err)
	}

	for i := range stages.Items {
		other := &stages.Items[i]
		if other.Name != stage.Name && other.DeletionTimestamp == nil && other.InCluster() &&
			other.Spec.Namespace == stage.Spec.Namespace {
			return true, nil
		}
	}

	return false, nil
}
