package kiosk

import (
	"context"

	"github.com/go-logr/logr"
	loftKioskApi "github.com/loft-sh/kiosk/pkg/apis/tenancy/v1alpha1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/common"
)

type SpaceManager interface {
	Create(name, account string) error
	Get(name string) (*loftKioskApi.Space, error)
	Delete(name string) error
}

type Space struct {
	Client client.Client
	Log    logr.Logger
}

func InitSpace(client client.Client) SpaceManager {
	return Space{
		Client: client,
		Log:    ctrl.Log.WithName("space-manager"),
	}
}

func (s Space) Create(name, account string) error {
	log := s.Log.WithValues("name", name)
	log.Info("creating loft kiosk space")
	space := &loftKioskApi.Space{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				util.TenantLabelName: account,
			},
		},
		Spec: loftKioskApi.SpaceSpec{
			Account: account,
		},
	}
	if err := s.Client.Create(context.Background(), space); err != nil {
		return err
	}
	log.Info("loft kiosk space is created")
	return nil
}

func (s Space) Get(name string) (*loftKioskApi.Space, error) {
	log := s.Log.WithValues("name", name)
	log.Info("getting loft kiosk space resource")
	space := &loftKioskApi.Space{}
	if err := s.Client.Get(context.Background(), types.NamespacedName{
		Name: name,
	}, space); err != nil {
		return nil, err
	}
	log.Info("loft kiosk space has been retrieved")
	return space, nil
}

func (s Space) Delete(name string) error {
	log := s.Log.WithValues("name", name)
	log.Info("deleting loft kiosk space")
	if err := s.Client.Delete(context.Background(), &loftKioskApi.Space{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
		},
	}, &client.DeleteOptions{
		GracePeriodSeconds: common.GetInt64P(0),
	}); err != nil {
		return err
	}
	log.Info("loft kiosk space is deleted")
	return nil
}
