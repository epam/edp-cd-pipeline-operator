package chain

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

type PutConfigMap struct {
	k8sClient client.Client
}

func NewPutConfigMap(k8sClient client.Client) *PutConfigMap {
	return &PutConfigMap{k8sClient: k8sClient}
}

func (h *PutConfigMap) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start putting ConfigMap", "configMapName", stage.Name)

	cm := &corev1.ConfigMap{}

	err := h.k8sClient.Get(
		ctx,
		types.NamespacedName{
			Namespace: stage.Namespace,
			Name:      stage.Name,
		},
		cm,
	)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	if err == nil {
		log.Info("ConfigMap already exists")

		return nil
	}

	cm = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stage.Name,
			Namespace: stage.Namespace,
		},
	}

	if err = controllerutil.SetControllerReference(stage, cm, h.k8sClient.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err = h.k8sClient.Create(ctx, cm); err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	log.Info("ConfigMap has been created")

	return nil
}
