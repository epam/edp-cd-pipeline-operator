package chain

import (
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/rbac"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("jenkins-job")

func nextServeOrNil(next handler.CdStageHandler, stage *v1alpha1.Stage) error {
	if next != nil {
		return next.ServeRequest(stage)
	}
	log.Info("handling of cd stage has been finished", "name", stage.Name)
	return nil
}

func CreateDefChain(client client.Client, triggerType string) handler.CdStageHandler {
	return getChain(client, triggerType)
}

func CreateDeleteChain(client client.Client) handler.CdStageHandler {
	log.Info("deletion chain is selected", "type", "auto deploy")
	log := log.WithName("delete-chain")
	return DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: client,
		log:    log.WithName("delete-environment-label-from-codebase-image-streams"),
		next: DeleteSpace{
			log:   log.WithName("delete-space"),
			space: kiosk.InitSpace(client),
		},
	}
}

func getChain(client client.Client, triggerType string) handler.CdStageHandler {
	space := kiosk.InitSpace(client)
	rbac := rbac.InitRbacManager(client)
	if consts.AutoDeployTriggerType == triggerType {
		log = log.WithName("create-chain").WithName("auto-deploy")
		return PutCodebaseImageStream{
			next: PutTenant{
				next: ConfigureRbac{
					next: PutJenkinsJob{
						client: client,
						next: PutEnvironmentLabelToCodebaseImageStreams{
							client: client,
							log:    log.WithName("put-environment-label-to-codebase-image-streams-chain"),
						},
						log: log.WithName("put-jenkins-job-chain"),
					},
					client: client,
					log:    log.WithName("configure-rbac"),
					rbac:   rbac,
				},
				client: client,
				log:    log.WithName("put-tenant"),
				space:  space,
			},
			client: client,
			log:    log.WithName("put-codebase-image-stream-chain"),
		}
	}
	log = log.WithName("create-chain").WithName("manual-deploy")
	return PutCodebaseImageStream{
		next: PutTenant{
			next: ConfigureRbac{
				next: PutJenkinsJob{
					client: client,
					log:    log.WithName("put-jenkins-job-chain"),
				},
				client: client,
				log:    log.WithName("configure-rbac"),
				rbac:   rbac,
			},
			client: client,
			log:    log.WithName("put-tenant"),
			space:  space,
		},
		client: client,
		log:    log.WithName("put-codebase-image-stream-chain"),
	}
}
