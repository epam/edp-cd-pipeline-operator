package util

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/helper"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

func GetCdPipeline(k8sClient client.Client, stage *cdPipeApi.Stage) (*cdPipeApi.CDPipeline, error) {
	ownerPipe := helper.GetOwnerReference(consts.CDPipelineKind, stage.GetOwnerReferences())
	if ownerPipe != nil {
		pipeline, err := cluster.GetCdPipeline(k8sClient, ownerPipe.Name, stage.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get owner pipeline : %w", err)
		}

		return pipeline, nil
	}

	pipeline, err := cluster.GetCdPipeline(k8sClient, stage.Spec.CdPipeline, stage.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	return pipeline, nil
}

func FindPreviousStageName(ctx context.Context, k8sClient client.Client, stage *cdPipeApi.Stage) (string, error) {
	if stage.IsFirst() {
		return "", errors.New("can't get previous stage from first stage")
	}

	stages := &cdPipeApi.StageList{}
	if err := k8sClient.List(
		ctx,
		stages,
		client.InNamespace(stage.Namespace),
		client.MatchingLabels{cdPipeApi.StageCdPipelineLabelName: stage.Spec.CdPipeline},
	); err != nil {
		return "", fmt.Errorf("failed to list stage names: %w", err)
	}

	//nolint
	//refactor
	for _, val := range stages.Items {
		if val.Spec.CdPipeline == stage.Spec.CdPipeline && val.Spec.Order == (stage.Spec.Order-1) {
			return val.Spec.Name, nil
		}
	}

	return "", errors.New("previous stage not found")
}

// GenerateNamespaceName generates namespace name based on stage name and namespace.
func GenerateNamespaceName(stage *cdPipeApi.Stage) string {
	return fmt.Sprintf("%s-%s", stage.Namespace, stage.Name)
}
