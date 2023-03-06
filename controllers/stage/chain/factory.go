package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

var log = ctrl.Log.WithName("stage")

const (
	putCodebaseImageStreamChain                   = "put-codebase-image-stream-chain"
	deleteEnvironmentLabelFromCodebaseImageStream = "delete-environment-label-from-codebase-image-streams"
	logKeyRegistryViewerRbac                      = "sa-registry-viewer-rbac"
	logKeyTenantAdminRbac                         = "tenant-admin-rbac"
	logKeyPutNamespace                            = "put-namespace"
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

// getDefChain returns a default chain of handlers for stage.
// nolint:funlen // it's a chain builder without any complex logic.
func getDefChain(c client.Client, triggerType string) handler.CdStageHandler {
	const (
		configureRbac      = "configure-rbac"
		putJenkinsJobChain = "put-jenkins-job-chain"
	)

	logger := ctrl.Log.WithName("create-chain")
	rbacManager := rbac.NewRbacManager(c, ctrl.Log.WithName("rbac-manager"))

	if consts.AutoDeployTriggerType == triggerType {
		logger.Info("Auto-deploy chain is selected")

		return PutCodebaseImageStream{
			next: DelegateNamespaceCreation{
				next: ConfigureJenkinsRbac{
					next: ConfigureRegistryViewerRbac{
						next: ConfigureTenantAdminRbac{
							client: c,
							log:    ctrl.Log.WithName(logKeyTenantAdminRbac),
							rbac:   rbacManager,
							next: PutJenkinsJob{
								client: c,
								next: RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
									client: c,
									log:    ctrl.Log.WithName("remove-labels-from-codebase-docker-streams-after-cd-pipeline-update"),
									next: DeleteEnvironmentLabelFromCodebaseImageStreams{
										client: c,
										log:    ctrl.Log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
										next: PutEnvironmentLabelToCodebaseImageStreams{
											client: c,
											log:    ctrl.Log.WithName("put-environment-label-to-codebase-image-streams"),
										},
									},
								},
								log: ctrl.Log.WithName(putJenkinsJobChain),
							},
						},
						client: c,
						log:    ctrl.Log.WithName(logKeyRegistryViewerRbac),
						rbac:   rbacManager,
					},
					client: c,
					log:    ctrl.Log.WithName(configureRbac),
					rbac:   rbacManager,
				},
				client: c,
				log:    ctrl.Log.WithName(logKeyPutNamespace),
			},
			client: c,
			log:    ctrl.Log.WithName(putCodebaseImageStreamChain),
		}
	}

	logger.Info("Manual-deploy chain is selected")

	return PutCodebaseImageStream{
		next: DelegateNamespaceCreation{
			next: ConfigureJenkinsRbac{
				next: ConfigureRegistryViewerRbac{
					client: c,
					log:    ctrl.Log.WithName(logKeyRegistryViewerRbac),
					rbac:   rbacManager,
					next: ConfigureTenantAdminRbac{
						client: c,
						log:    ctrl.Log.WithName(logKeyTenantAdminRbac),
						rbac:   rbacManager,
						next: PutJenkinsJob{
							client: c,
							log:    ctrl.Log.WithName(putJenkinsJobChain),
							next: DeleteEnvironmentLabelFromCodebaseImageStreams{
								client: c,
								log:    ctrl.Log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
							},
						},
					},
				},
				client: c,
				log:    ctrl.Log.WithName(configureRbac),
				rbac:   rbacManager,
			},
			client: c,
			log:    ctrl.Log.WithName(logKeyPutNamespace),
		},
		client: c,
		log:    ctrl.Log.WithName(putCodebaseImageStreamChain),
	}
}

// getTektonDeleteChain returns a chain of handlers for tekton flow.
// nolint:funlen // it's a chain builder without any complex logic.
func getTektonChain(c client.Client, triggerType string) handler.CdStageHandler {
	logger := ctrl.Log.WithName("create-chain")
	rbacManager := rbac.NewRbacManager(c, ctrl.Log.WithName("rbac-manager"))

	logger.Info("Tekton chain is selected")

	if consts.AutoDeployTriggerType == triggerType {
		logger.Info("Auto-deploy chain is selected")

		return PutCodebaseImageStream{
			next: DelegateNamespaceCreation{
				client: c,
				log:    ctrl.Log.WithName(logKeyPutNamespace),
				next: RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
					client: c,
					log:    ctrl.Log.WithName("remove-labels-from-codebase-docker-streams-after-cd-pipeline-update"),
					next: DeleteEnvironmentLabelFromCodebaseImageStreams{
						client: c,
						log:    ctrl.Log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
						next: PutEnvironmentLabelToCodebaseImageStreams{
							client: c,
							log:    ctrl.Log.WithName("put-environment-label-to-codebase-image-streams-chain"),
							next: ConfigureRegistryViewerRbac{
								client: c,
								log:    ctrl.Log.WithName(logKeyRegistryViewerRbac),
								rbac:   rbacManager,
								next: ConfigureTenantAdminRbac{
									client: c,
									log:    ctrl.Log.WithName(logKeyTenantAdminRbac),
									rbac:   rbacManager,
								},
							},
						},
					},
				},
			},
			client: c,
			log:    ctrl.Log.WithName(putCodebaseImageStreamChain),
		}
	}

	logger.Info("Manual-deploy chain is selected")

	return PutCodebaseImageStream{
		next: DelegateNamespaceCreation{
			client: c,
			log:    ctrl.Log.WithName(logKeyPutNamespace),
			next: DeleteEnvironmentLabelFromCodebaseImageStreams{
				client: c,
				log:    ctrl.Log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
				next: ConfigureRegistryViewerRbac{
					client: c,
					log:    ctrl.Log.WithName(logKeyRegistryViewerRbac),
					rbac:   rbacManager,
					next: ConfigureTenantAdminRbac{
						client: c,
						log:    ctrl.Log.WithName(logKeyTenantAdminRbac),
						rbac:   rbacManager,
					},
				},
			},
		},
		client: c,
		log:    ctrl.Log.WithName(putCodebaseImageStreamChain),
	}
}

func createDefDeleteChain(c client.Client) handler.CdStageHandler {
	logger := ctrl.Log.WithName("delete-chain")

	logger.Info("Default delete chain is selected")

	return DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: c,
		log:    ctrl.Log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
		next: DelegateNamespaceDeletion{
			client: c,
			log:    ctrl.Log.WithName("delete-namespace"),
			next: DeleteRegistryViewerRbac{
				client: c,
				log:    ctrl.Log.WithName("delete-registry-viewer-rbac"),
			},
		},
	}
}

func getTektonDeleteChain(c client.Client) handler.CdStageHandler {
	logger := ctrl.Log.WithName("delete-chain")
	logger.Info("Tekton deletion chain is selected")

	return DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: c,
		log:    ctrl.Log.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
		next: DelegateNamespaceDeletion{
			client: c,
			log:    ctrl.Log.WithName("delete-namespace"),
			next: DeleteRegistryViewerRbac{
				client: c,
				log:    ctrl.Log.WithName("delete-registry-viewer-rbac"),
			},
		},
	}
}
