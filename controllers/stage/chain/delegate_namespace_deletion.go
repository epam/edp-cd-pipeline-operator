package chain

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

// DelegateNamespaceDeletion is a stage chain element that decides whether to delete a namespace or project.
type DelegateNamespaceDeletion struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

// ServeRequest creates for kubernetes platform DeleteNamespace or DeleteSpace if kiosk is enabled.
// For platform openshift it creates DeleteOpenshiftProject.
// The decision is made based on the environment variable PLATFORM_TYPE.
// By default, it creates DeleteOpenshiftProject.
// If the namespace is not managed by the operator, it creates Skip chain element.
func (c DelegateNamespaceDeletion) ServeRequest(stage *cdPipeApi.Stage) error {
	logger := c.log.WithValues("stage name", stage.Name)

	if !platform.ManageNamespace() {
		logger.Info("Namespace is not managed by the operator")

		return nextServeOrNil(Skip{
			next: c.next,
			log:  c.log,
		}, stage)
	}

	if platform.IsKubernetes() {
		logger.Info("Platform is kubernetes")

		if platform.KioskEnabled() {
			logger.Info("Kiosk is enabled")

			return nextServeOrNil(DeleteSpace{
				next:  c.next,
				space: kiosk.InitSpace(c.client),
				log:   c.log,
			}, stage)
		}

		if platform.CapsuleEnabled() {
			logger.Info("Capsule is enabled")
		} else {
			logger.Info("None of multi-tenancy engines is enabled")
		}

		return nextServeOrNil(DeleteNamespace(c), stage)
	}

	logger.Info("Platform is openshift")

	return nextServeOrNil(DeleteOpenshiftProject(c), stage)
}
