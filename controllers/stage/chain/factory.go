package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/multiclusterclient"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
)

var log = ctrl.Log.WithName("stage")

const (
	putCodebaseImageStreamChain                   = "put-codebase-image-stream-chain"
	deleteEnvironmentLabelFromCodebaseImageStream = "delete-env-label-cis"
	logKeyRegistryViewerRbac                      = "sa-registry-viewer-rbac"
	logKeyTenantAdminRbac                         = "tenant-admin-rbac"
	logKeyPutNamespace                            = "put-namespace"
	logKeyManageSecretsRBAC                       = "manage-secrets-rbac"
)

// multiClusterClient is a internalClient for external cluster connection.
// It will connect to internal cluster if stage.Spec.ClusterName is "in-cluster".
type multiClusterClient client.Client

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

func CreateChain(ctx context.Context, c client.Client, stage *cdPipeApi.Stage) (handler.CdStageHandler, error) {
	logger := ctrl.Log.WithName("create-chain")

	multiClusterCl, err := multiclusterclient.NewClientProvider(c).GetClusterClient(ctx, stage.Namespace, stage.Spec.ClusterName, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster internalClient: %w", err)
	}

	rbacManager := rbac.NewRbacManager(multiClusterCl, ctrl.Log.WithName("rbac-manager"))

	logger.Info("Tekton chain is selected")

	return PutCodebaseImageStream{
		next: DelegateNamespaceCreation{
			client: multiClusterCl,
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
							log:  ctrl.Log.WithName(logKeyRegistryViewerRbac),
							rbac: rbacManager,
							next: ConfigureTenantAdminRbac{
								log:  ctrl.Log.WithName(logKeyTenantAdminRbac),
								rbac: rbacManager,
								next: ConfigureSecretManager{
									multiClusterClient: multiClusterCl,
									internalClient:     c,
									log:                ctrl.Log.WithName(logKeyManageSecretsRBAC),
								},
							},
						},
					},
				},
			},
		},
		client: c,
		log:    ctrl.Log.WithName(putCodebaseImageStreamChain),
	}, nil
}

func CreateDeleteChain(ctx context.Context, c client.Client, stage *cdPipeApi.Stage) (handler.CdStageHandler, error) {
	logger := ctrl.LoggerFrom(ctx)

	logger.Info("Delete chain is selected")

	multiClusterCl, err := multiclusterclient.NewClientProvider(c).GetClusterClient(ctx, stage.Namespace, stage.Spec.ClusterName, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster internalClient: %w", err)
	}

	return DeleteEnvironmentLabelFromCodebaseImageStreams{
		client: c,
		log:    logger.WithName(deleteEnvironmentLabelFromCodebaseImageStream),
		next: DelegateNamespaceDeletion{
			multiClusterClient: multiClusterCl,
			log:                logger.WithName("delete-namespace"),
			next: DeleteRegistryViewerRbac{
				multiClusterCl: multiClusterCl,
				log:            logger.WithName("delete-registry-viewer-rbac"),
			},
		},
	}, nil
}
