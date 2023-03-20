package kiosk

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

const crdNameKey = "name"

type SpaceManager interface {
	Create(name, account string) error
	Get(name string) (*unstructured.Unstructured, error)
	Delete(name string) error
}

type Space struct {
	Client client.Client
	Log    logr.Logger
}

func InitSpace(c client.Client) SpaceManager {
	return Space{
		Client: c,
		Log:    ctrl.Log.WithName("space-manager"),
	}
}

func (s Space) Create(name, account string) error {
	log := s.Log.WithValues(crdNameKey, name)
	log.Info("creating loft kiosk space")

	space := &unstructured.Unstructured{}
	space.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "tenancy.kiosk.sh/v1alpha1",
		"metadata": map[string]interface{}{
			crdNameKey: name,
			"labels": map[string]interface{}{
				util.TenantLabelName: account,
			},
		},
		"spec": map[string]interface{}{
			"account": account,
		},
	}

	if err := s.Client.Create(context.Background(), space); err != nil {
		return fmt.Errorf("failed to create loft kiosk space: %w", err)
	}

	log.Info("loft kiosk space is created")

	return nil
}

func (s Space) Get(name string) (*unstructured.Unstructured, error) {
	log := s.Log.WithValues(crdNameKey, name)
	log.Info("getting loft kiosk space resource")

	space := &unstructured.Unstructured{}
	space.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "tenancy.kiosk.sh/v1alpha1",
	}

	if err := s.Client.Get(context.Background(), types.NamespacedName{
		Name: name,
	}, space); err != nil {
		return nil, fmt.Errorf("failed to retrieve loft kiosk space: %w", err)
	}

	log.Info("loft kiosk space has been retrieved")

	return space, nil
}

func (s Space) Delete(name string) error {
	log := s.Log.WithValues(crdNameKey, name)
	log.Info("deleting loft kiosk space")

	space := &unstructured.Unstructured{}
	space.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "tenancy.kiosk.sh/v1alpha1",
		"metadata": map[string]interface{}{
			crdNameKey: name,
		},
	}

	if err := s.Client.Delete(
		context.Background(),
		space,
		&client.DeleteOptions{
			GracePeriodSeconds: pointer.Int64(0),
		},
	); err != nil {
		return fmt.Errorf("failed to delete loft kiosk space: %w", err)
	}

	log.Info("loft kiosk space is deleted")

	return nil
}
