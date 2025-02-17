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
)

type DeleteNamespace struct {
	multiClusterClient multiClusterClient
}

func (h DeleteNamespace) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	l := ctrl.LoggerFrom(ctx).WithValues("namespace", stage.Spec.Namespace)

	if err := h.multiClusterClient.Get(ctx, client.ObjectKey{Name: stage.Spec.Namespace}, &corev1.Namespace{}); err != nil {
		if apierrors.IsNotFound(err) {
			l.Info("Namespace has already been deleted")

			return nil
		}

		return fmt.Errorf("failed to get namespace: %w", err)
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
