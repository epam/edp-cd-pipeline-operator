package util

import (
	"context"
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

func GetCdPipeline(k8sClient client.Client, stage *cdPipeApi.Stage) (*cdPipeApi.CDPipeline, error) {
	ownerPipe := helper.GetOwnerReference(consts.CDPipelineKind, stage.GetOwnerReferences())
	if ownerPipe != nil {
		return cluster.GetCdPipeline(k8sClient, ownerPipe.Name, stage.Namespace)
	}

	return cluster.GetCdPipeline(k8sClient, stage.Spec.CdPipeline, stage.Namespace)
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
		client.MatchingLabels{cdPipeApi.CodebaseTypeLabelName: stage.Spec.CdPipeline},
	); err != nil {
		return "", err
	}

	for _, val := range stages.Items {
		if val.Spec.CdPipeline == stage.Spec.CdPipeline && val.Spec.Order == (stage.Spec.Order-1) {
			return val.Spec.Name, nil
		}
	}

	return "", errors.New("previous stage not found")
}
