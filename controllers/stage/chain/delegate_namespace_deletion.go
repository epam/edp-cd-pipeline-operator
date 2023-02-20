package chain

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/kiosk"
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
func (c DelegateNamespaceDeletion) ServeRequest(stage *cdPipeApi.Stage) error {
	if platform.IsKubernetes() {
		if platform.KioskEnabled() {
			return nextServeOrNil(DeleteSpace{
				next:  c.next,
				space: kiosk.InitSpace(c.client),
				log:   c.log,
			}, stage)
		}

		return nextServeOrNil(DeleteNamespace(c), stage)
	}

	return nextServeOrNil(DeleteOpenshiftProject(c), stage)
}
