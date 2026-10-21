package chain

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
)

type PutNamespace struct {
	client multiClusterClient
}

// Stage target namespaces are arbitrary, so this rule is cluster-scoped.
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;create

func (h PutNamespace) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	l := ctrl.LoggerFrom(context.Background())

	l.Info("Creating namespace")

	ns := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: stage.Spec.Namespace,
			Labels: map[string]string{
				util.TenantLabelName: stage.Namespace,
			},
		},
	}
	if err := h.client.Create(ctx, ns); err != nil {
		if apierrors.IsAlreadyExists(err) {
			l.Info("Namespace already exists")

			if !util.OwnershipEnforced(stage) {
				return nil
			}

			existing := &v1.Namespace{}
			if err = h.client.Get(ctx, client.ObjectKey{Name: stage.Spec.Namespace}, existing); err != nil {
				return fmt.Errorf("failed to get namespace: %w", err)
			}

			return util.CheckNamespaceOwnership(existing, stage.Namespace)
		}

		return fmt.Errorf("failed to create namespace: %w", err)
	}

	l.Info("Namespace has been created")

	return nil
}
