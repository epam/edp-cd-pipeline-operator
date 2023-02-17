package chain

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/rbac"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

var log = ctrl.Log.WithName("stage")

const (
	configureRbac                                 = "configure-rbac"
	putJenkinsJobChain                            = "put-jenkins-job-chain"
	putCodebaseImageStreamChain                   = "put-codebase-image-stream-chain"
	deleteEnvironmentLabelFromCodebaseImageStream = "delete-environment-label-from-codebase-image-streams"
	createChain                                   = "create-chain"
)

func nextServeOrNil(next handler.CdStageHandler, stage *cdPipeApi.Stage) error {
	if next != nil {
		if err := next.ServeRequest(stage); err != nil {
			return fmt.Errorf("failed to serve request: %w", err)
		}

		return nil
	}

	log.Info("handling of cd stage has been finished", "name", stage.Name)

	return nil
}

func CreateChain(ctx context.Context, c client.Client, namespace, triggerType string) handler.CdStageHandler {
	if !cluster.JenkinsEnabled(ctx, c, namespace, log) {
		return getTektonChain(c, triggerType)
	}

	if kioskEnabled() {
		return getKioskChain(c, triggerType)
	}

	return getDefChain(c, triggerType)
}

func kioskEnabled() bool {
	if enabled := os.Getenv("KIOSK_ENABLED"); enabled == "" {
		return false
	}

	enabled, _ := strconv.ParseBool(os.Getenv("KIOSK_ENABLED"))

	return enabled
}

func CreateDeleteChain(ctx context.Context, c client.Client, namespace string) handler.CdStageHandler {
	if !cluster.JenkinsEnabled(ctx, c, namespace, log) {
		return getTektonDeleteChain(c)
	}

	if kioskEnabled() {
		return createKioskDeleteChain(c)
	}

	return createDefDeleteChain(c)
}

func createKioskDeleteChain(c client.Client) handler.CdStageHandler {
	log.Info("deletion chain is selected", "kiosk", "enabled", "type", "auto deploy")
	logger := log.WithName("delete-chain")

	return DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: c,
		log:    logger.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
		next: DeleteSpace{
			log:   logger.WithName("delete-space"),
			space: kiosk.InitSpace(c),
		},
	}
}

func createDefDeleteChain(c client.Client) handler.CdStageHandler {
	log.Info("deletion chain is selected", "kiosk", "disabled", "type", "auto deploy")
	logger := log.WithName("delete-chain")

	return DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: c,
		log:    log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
		next: DelegateNamespaceDeletion{
			client: c,
			log:    logger.WithName("delete-namespace"),
		},
	}
}

func getDefChain(c client.Client, triggerType string) handler.CdStageHandler {
	rbacManager := rbac.NewRbacManager(c, log.WithName("rbac-manager"))

	if consts.AutoDeployTriggerType == triggerType {
		logger := log.WithName(createChain).WithName("auto-deploy")

		return PutCodebaseImageStream{
			next: DelegateNamespaceCreation{
				next: ConfigureRbac{
					next: PutJenkinsJob{
						client: c,
						next: RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
							client: c,
							log:    logger.WithName("remove-labels-from-codebase-docker-streams-after-cd-pipeline-update"),
							next: DeleteEnvironmentLabelFromCodebaseImageStreams{
								client: c,
								log:    logger.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
								next: PutEnvironmentLabelToCodebaseImageStreams{
									client: c,
									log:    logger.WithName("put-environment-label-to-codebase-image-streams-chain"),
								},
							},
						},
						log: logger.WithName(putJenkinsJobChain),
					},
					client: c,
					log:    logger.WithName(configureRbac),
					rbac:   rbacManager,
				},
				client: c,
				log:    logger.WithName("put-namespace"),
			},
			client: c,
			log:    logger.WithName(putCodebaseImageStreamChain),
		}
	}

	logger := log.WithName(createChain).WithName("manual-deploy")

	return PutCodebaseImageStream{
		next: DelegateNamespaceCreation{
			next: ConfigureRbac{
				next: PutJenkinsJob{
					client: c,
					log:    logger.WithName(putJenkinsJobChain),
					next: DeleteEnvironmentLabelFromCodebaseImageStreams{
						client: c,
						log:    logger.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
					},
				},
				client: c,
				log:    logger.WithName(configureRbac),
				rbac:   rbacManager,
			},
			client: c,
			log:    logger.WithName("put-namespace"),
		},
		client: c,
		log:    logger.WithName(putCodebaseImageStreamChain),
	}
}

func getKioskChain(c client.Client, triggerType string) handler.CdStageHandler {
	space := kiosk.InitSpace(c)
	rbacManager := rbac.NewRbacManager(c, log.WithName("rbac-manager"))

	if consts.AutoDeployTriggerType == triggerType {
		return getAutoDeployPutCodebaseImageStream(c, log.WithName(createChain).WithName("auto-deploy"), rbacManager, space)
	}

	return getManualDeployPutCodebaseImageStream(c, log.WithName(createChain).WithName("manual-deploy"), rbacManager, space)
}

func getManualDeployPutCodebaseImageStream(c client.Client, logger logr.Logger, rbacManager rbac.Manager, space kiosk.SpaceManager) PutCodebaseImageStream {
	return PutCodebaseImageStream{
		next: PutKioskSpace{
			next: ConfigureRbac{
				next: PutJenkinsJob{
					client: c,
					log:    logger.WithName(putJenkinsJobChain),
					next: DeleteEnvironmentLabelFromCodebaseImageStreams{
						client: c,
						log:    logger.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
					},
				},
				client: c,
				log:    logger.WithName(configureRbac),
				rbac:   rbacManager,
			},
			client: c,
			log:    logger.WithName("put-tenant"),
			space:  space,
		},
		client: c,
		log:    logger.WithName(putCodebaseImageStreamChain),
	}
}

func getAutoDeployPutCodebaseImageStream(c client.Client, logger logr.Logger, rbacManager rbac.Manager, space kiosk.SpaceManager) PutCodebaseImageStream {
	return PutCodebaseImageStream{
		next: PutKioskSpace{
			next: ConfigureRbac{
				next: PutJenkinsJob{
					client: c,
					next: RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
						client: c,
						log:    logger.WithName("remove-labels-from-codebase-docker-streams-after-cd-pipeline-update"),
						next: DeleteEnvironmentLabelFromCodebaseImageStreams{
							client: c,
							log:    logger.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
							next: PutEnvironmentLabelToCodebaseImageStreams{
								client: c,
								log:    logger.WithName("put-environment-label-to-codebase-image-streams-chain"),
							},
						},
					},
					log: logger.WithName(putJenkinsJobChain),
				},
				client: c,
				log:    logger.WithName(configureRbac),
				rbac:   rbacManager,
			},
			client: c,
			log:    logger.WithName("put-tenant"),
			space:  space,
		},
		client: c,
		log:    logger.WithName(putCodebaseImageStreamChain),
	}
}

func getTektonChain(c client.Client, triggerType string) handler.CdStageHandler {
	log.Info("tekton chain is selected")

	if consts.AutoDeployTriggerType == triggerType {
		return PutCodebaseImageStream{
			next: RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
				client: c,
				log:    log.WithName("remove-labels-from-codebase-docker-streams-after-cd-pipeline-update"),
				next: DeleteEnvironmentLabelFromCodebaseImageStreams{
					client: c,
					log:    log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
					next: PutEnvironmentLabelToCodebaseImageStreams{
						client: c,
						log:    log.WithName("put-environment-label-to-codebase-image-streams-chain"),
					},
				},
			},
			client: c,
			log:    log.WithName(putCodebaseImageStreamChain),
		}
	}

	return PutCodebaseImageStream{
		next: DeleteEnvironmentLabelFromCodebaseImageStreams{
			client: c,
			log:    log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
		},
		client: c,
		log:    log.WithName(putCodebaseImageStreamChain),
	}
}

func getTektonDeleteChain(c client.Client) handler.CdStageHandler {
	log.Info("tekton deletion chain is selected")

	return DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: c,
		log:    log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
		next: DelegateNamespaceDeletion{
			client: c,
			log:    log.WithName("delete-namespace"),
		},
	}
}
