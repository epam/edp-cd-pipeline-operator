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

// multiClusterClient is a internalClient for external cluster connection.
// It will connect to internal cluster if stage.Spec.ClusterName is "in-cluster".
type multiClusterClient client.Client

func CreateChain(ctx context.Context, c client.Client, stage *cdPipeApi.Stage) (handler.CdStageHandler, error) {
	multiClusterCl, err := multiclusterclient.NewClientProvider(c).GetClusterClient(ctx, stage.Namespace, stage.Spec.ClusterName, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster internalClient: %w", err)
	}

	rbacManager := rbac.NewRbacManager(multiClusterCl, ctrl.Log.WithName("rbac-manager"))

	ch := &chain{}
	ch.Use(
		PutCodebaseImageStream{
			client: c,
		},
		DelegateNamespaceCreation{
			client: multiClusterCl,
		},
		RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
			client: c,
		},
		DeleteEnvironmentLabelFromCodebaseImageStreams{
			client: c,
		},
		PutEnvironmentLabelToCodebaseImageStreams{
			client: c,
		},
		ConfigureRegistryViewerRbac{
			rbac: rbacManager,
		},
		ConfigureTenantAdminRbac{
			rbac: rbacManager,
		},
		ConfigureSecretManager{
			multiClusterClient: multiClusterCl,
			internalClient:     c,
		},
	)

	return ch, nil
}

func CreateDeleteChain(ctx context.Context, c client.Client, stage *cdPipeApi.Stage) (handler.CdStageHandler, error) {
	log := ctrl.LoggerFrom(ctx)
	ch := &chain{}

	ch.Use(
		DeleteEnvironmentLabelFromCodebaseImageStreams{
			client: c,
		},
	)

	multiClusterCl, err := multiclusterclient.NewClientProvider(c).GetClusterClient(ctx, stage.Namespace, stage.Spec.ClusterName, client.Options{})
	if err != nil {
		log.Error(err, "Failed to get cluster internalClient. Skipping namespace deletion.")
		return ch, nil
	}

	ch.Use(
		DelegateNamespaceDeletion{
			multiClusterClient: multiClusterCl,
		},
		DeleteRegistryViewerRbac{
			multiClusterCl: multiClusterCl,
		},
	)

	return ch, nil
}
