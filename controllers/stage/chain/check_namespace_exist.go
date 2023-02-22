package chain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

// CheckNamespaceExist checks if namespace exists.
type CheckNamespaceExist struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

// ServeRequest serves request to check if namespace exists.
func (h CheckNamespaceExist) ServeRequest(stage *cdPipeApi.Stage) error {
	name := util.GenerateNamespaceName(stage)

	h.log.Info("Checking existence of namespace", crNameLogKey, name)

	if err := h.client.Get(context.TODO(), types.NamespacedName{
		Name: name,
	}, &v1.Namespace{}); err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("namespace %s doesn't exist", name)
		}

		return fmt.Errorf("failed to check existence of %s namespace: %w", name, err)
	}

	return nextServeOrNil(h.next, stage)
}
