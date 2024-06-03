package chain

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

// DelegateNamespaceCreation is a stage chain element that decides whether to create a namespace or project.
type DelegateNamespaceCreation struct {
	client multiClusterClient
}

// ServeRequest is responsible for delegating the creation of a namespace or project
// based on the platform and the configuration of the stage.
func (c DelegateNamespaceCreation) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	logger := ctrl.LoggerFrom(ctx)

	if !platform.ManageNamespace() {
		logger.Info("Namespace is not managed by the operator")

		return CheckNamespaceExist(c).ServeRequest(ctx, stage)
	}

	if platform.IsKubernetes() {
		logger.Info("Platform is kubernetes")

		if !stage.InCluster() {
			logger.Info("Stage is not in cluster. Skip multi-tenancy engines")

			return PutNamespace(c).ServeRequest(ctx, stage)
		}

		if platform.CapsuleEnabled() {
			logger.Info("Capsule is enabled")
		} else {
			logger.Info("None of multi-tenancy engines is enabled")
		}

		return PutNamespace(c).ServeRequest(ctx, stage)
	}

	logger.Info("Platform is openshift")

	return PutOpenshiftProject(c).ServeRequest(ctx, stage)
}
