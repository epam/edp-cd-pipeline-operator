package chain

import (
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/rbac"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

var log = ctrl.Log.WithName("jenkins-job")

func nextServeOrNil(next handler.CdStageHandler, stage *v1alpha1.Stage) error {
	if next != nil {
		return next.ServeRequest(stage)
	}
	log.Info("handling of cd stage has been finished", "name", stage.Name)
	return nil
}

func CreateChain(client client.Client, triggerType string) handler.CdStageHandler {
	if kioskEnabled() {
		return getKioskChain(client, triggerType)
	}
	return getDefChain(client, triggerType)
}

func kioskEnabled() bool {
	if enabled := os.Getenv("KIOSK_ENABLED"); enabled == "" {
		return false
	}

	enabled, _ := strconv.ParseBool(os.Getenv("KIOSK_ENABLED"))
	return enabled
}

func CreateDeleteChain(client client.Client) handler.CdStageHandler {
	if kioskEnabled() {
		return createKioskDeleteChain(client)
	}
	return createDefDeleteChain(client)
}

func createKioskDeleteChain(client client.Client) handler.CdStageHandler {
	log.Info("deletion chain is selected", "kiosk", "enabled", "type", "auto deploy")
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

func createDefDeleteChain(client client.Client) handler.CdStageHandler {
	log.Info("deletion chain is selected", "kiosk", "disabled", "type", "auto deploy")
	log := log.WithName("delete-chain")
	return DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: client,
		log:    log.WithName("delete-environment-label-from-codebase-image-streams"),
		next: DeleteNamespace{
			client: client,
			log:    log.WithName("delete-namespace"),
		},
	}
}

func getDefChain(client client.Client, triggerType string) handler.CdStageHandler {
	rbac := rbac.InitRbacManager(client)
	if consts.AutoDeployTriggerType == triggerType {
		log = log.WithName("create-chain").WithName("auto-deploy")
		return PutCodebaseImageStream{
			next: PutNamespace{
				next: ConfigureRbac{
					next: PutJenkinsJob{
						client: client,
						next: RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
							client: client,
							log:    log.WithName("remove-labels-from-codebase-docker-streams-after-cd-pipeline-update"),
							next: DeleteEnvironmentLabelFromCodebaseImageStreams{
								client: client,
								log:    log.WithName("delete-environment-label-from-codebase-image-streams"),
								next: PutEnvironmentLabelToCodebaseImageStreams{
									client: client,
									log:    log.WithName("put-environment-label-to-codebase-image-streams-chain"),
								},
							},
						},
						log: log.WithName("put-jenkins-job-chain"),
					},
					client: client,
					log:    log.WithName("configure-rbac"),
					rbac:   rbac,
				},
				client: client,
				log:    log.WithName("put-namespace"),
			},
			client: client,
			log:    log.WithName("put-codebase-image-stream-chain"),
		}
	}
	log = log.WithName("create-chain").WithName("manual-deploy")
	return PutCodebaseImageStream{
		next: PutNamespace{
			next: ConfigureRbac{
				next: PutJenkinsJob{
					client: client,
					log:    log.WithName("put-jenkins-job-chain"),
					next: DeleteEnvironmentLabelFromCodebaseImageStreams{
						client: client,
						log:    log.WithName("delete-environment-label-from-codebase-image-streams"),
					},
				},
				client: client,
				log:    log.WithName("configure-rbac"),
				rbac:   rbac,
			},
			client: client,
			log:    log.WithName("put-namespace"),
		},
		client: client,
		log:    log.WithName("put-codebase-image-stream-chain"),
	}
}

func getKioskChain(client client.Client, triggerType string) handler.CdStageHandler {
	space := kiosk.InitSpace(client)
	rbac := rbac.InitRbacManager(client)
	if consts.AutoDeployTriggerType == triggerType {
		log = log.WithName("create-chain").WithName("auto-deploy")
		return PutCodebaseImageStream{
			next: PutKioskSpace{
				next: ConfigureRbac{
					next: PutJenkinsJob{
						client: client,
						next: RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
							client: client,
							log:    log.WithName("remove-labels-from-codebase-docker-streams-after-cd-pipeline-update"),
							next: DeleteEnvironmentLabelFromCodebaseImageStreams{
								client: client,
								log:    log.WithName("delete-environment-label-from-codebase-image-streams"),
								next: PutEnvironmentLabelToCodebaseImageStreams{
									client: client,
									log:    log.WithName("put-environment-label-to-codebase-image-streams-chain"),
								},
							},
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
		next: PutKioskSpace{
			next: ConfigureRbac{
				next: PutJenkinsJob{
					client: client,
					log:    log.WithName("put-jenkins-job-chain"),
					next: DeleteEnvironmentLabelFromCodebaseImageStreams{
						client: client,
						log:    log.WithName("delete-environment-label-from-codebase-image-streams"),
					},
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
