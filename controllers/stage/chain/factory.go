package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/rbac"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

var log = ctrl.Log.WithName("stage")

const (
	putCodebaseImageStreamChain                   = "put-codebase-image-stream-chain"
	deleteEnvironmentLabelFromCodebaseImageStream = "delete-environment-label-from-codebase-image-streams"
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

	return getDefChain(c, triggerType)
}

func CreateDeleteChain(ctx context.Context, c client.Client, namespace string) handler.CdStageHandler {
	if !cluster.JenkinsEnabled(ctx, c, namespace, log) {
		return getTektonDeleteChain(c)
	}

	return createDefDeleteChain(c)
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

// getDefChain returns a default chain of handlers for stage.
// nolint:funlen // it's a chain builder without any complex logic.
func getDefChain(c client.Client, triggerType string) handler.CdStageHandler {
	const (
		configureRbac      = "configure-rbac"
		putJenkinsJobChain = "put-jenkins-job-chain"
		createChain        = "create-chain"
	)

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
