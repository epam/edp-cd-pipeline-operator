package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

type PutNamespace struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

func (h PutNamespace) ServeRequest(stage *cdPipeApi.Stage) error {
	l := h.log.WithValues("namespace", stage.Spec.Namespace)
	ctx := ctrl.LoggerInto(context.TODO(), l)

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

			return nil
		}

		return fmt.Errorf("failed to create namespace: %w", err)
	}

	l.Info("Namespace has been created")

	return nextServeOrNil(h.next, stage)
}
