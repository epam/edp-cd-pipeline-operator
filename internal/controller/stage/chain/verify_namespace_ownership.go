package chain

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
)

// VerifyNamespaceOwnership refuses to provision into an existing namespace that is not owned by the Stage tenant.
type VerifyNamespaceOwnership struct {
	client multiClusterClient
}

// ServeRequest returns util.ErrNamespaceNotOwned if the target namespace exists and is not labelled with the tenant.
// An absent namespace is allowed: it is created later in the chain with the tenant label.
func (h VerifyNamespaceOwnership) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	if !util.OwnershipEnforced(stage) {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	name := stage.Spec.Namespace

	ns := &corev1.Namespace{}
	if err := h.client.Get(ctx, types.NamespacedName{Name: name}, ns); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to get namespace %s: %w", name, err)
	}

	log.Info("Verifying ownership of namespace", "name", name)

	return util.CheckNamespaceOwnership(ns, stage.Namespace)
}
