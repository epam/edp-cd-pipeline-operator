package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
)

type DeleteNamespace struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

func (h DeleteNamespace) ServeRequest(stage *cdPipeApi.Stage) error {
	name := fmt.Sprintf("%v-%v", stage.Namespace, stage.Name)
	if err := h.delete(name); err != nil {
		return errors.Wrapf(err, "unable to delete %v namespace", name)
	}
	return nextServeOrNil(h.next, stage)
}

func (h DeleteNamespace) delete(name string) error {
	log := h.log.WithValues("name", name)
	log.Info("trying to delete namespace")
	ns := &v1.Namespace{}
	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Name: name,
	}, ns); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("namespace doesn't exist")
			return nil
		}
		return err
	}

	if err := h.client.Delete(context.TODO(), ns); err != nil {
		return err
	}
	log.Info("namespace has been deleted")
	return nil
}
