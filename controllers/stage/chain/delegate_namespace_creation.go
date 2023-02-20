package chain

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

// DelegateNamespaceCreation is a stage chain element that decides whether to create a namespace, kiosk space or project.
type DelegateNamespaceCreation struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

// ServeRequest creates for kubernetes platform PutNamespace or PutKioskSpace if the kiosk is enabled.
// For platform openshift it creates PutOpenshiftProject.
// The decision is made based on the environment variable PLATFORM_TYPE.
// By default, it creates PutOpenshiftProject.
func (c DelegateNamespaceCreation) ServeRequest(stage *cdPipeApi.Stage) error {
	if platform.IsKubernetes() {
		if platform.KioskEnabled() {
			return nextServeOrNil(PutKioskSpace{
				next:   c.next,
				space:  kiosk.InitSpace(c.client),
				client: c.client,
				log:    c.log,
			}, stage)
		}

		return nextServeOrNil(PutNamespace(c), stage)
	}

	return nextServeOrNil(PutOpenshiftProject(c), stage)
}
