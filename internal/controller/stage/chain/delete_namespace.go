package chain

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
)

type DeleteNamespace struct {
	multiClusterClient multiClusterClient
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;delete

func (h DeleteNamespace) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	l := ctrl.LoggerFrom(ctx).WithValues("namespace", stage.Spec.Namespace)

	ns := &corev1.Namespace{}
	if err := h.multiClusterClient.Get(
		ctx,
		client.ObjectKey{Name: stage.Spec.Namespace},
		ns,
	); err != nil {
		if apierrors.IsNotFound(err) {
			l.Info("Namespace has already been deleted")

			return nil
		}

		return fmt.Errorf("failed to get namespace: %w", err)
	}

	if util.OwnershipEnforced(stage) {
		if err := util.CheckNamespaceOwnership(ns, stage.Namespace); err != nil {
			l.Info("Namespace is not owned by the tenant, skipping deletion", "reason", err.Error())

			return nil
		}
	}

	if err := client.IgnoreNotFound(
		h.multiClusterClient.Delete(
			ctx,
			&corev1.Namespace{
				ObjectMeta: metaV1.ObjectMeta{
					Name: stage.Spec.Namespace,
				},
			},
		),
	); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	l.Info("Namespace has been deleted")

	return nil
}
