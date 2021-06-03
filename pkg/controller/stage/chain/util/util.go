package util

import (
	"context"
	"fmt"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"github.com/pkg/errors"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCdPipeline(client k8sClient.Client, stage *v1alpha1.Stage) (*v1alpha1.CDPipeline, error) {
	ownerPipe := helper.GetOwnerReference(consts.CDPipelineKind, stage.GetOwnerReferences())
	if ownerPipe != nil {
		return cluster.GetCdPipeline(client, ownerPipe.Name, stage.Namespace)
	}
	return cluster.GetCdPipeline(client, stage.Spec.CdPipeline, stage.Namespace)
}

func FindPreviousStage(client k8sClient.Client, currentOrder int, pipelineName, ns string) (*v1alpha1.Stage, error) {
	stages, err := findStagesRelatedToPipeline(client, pipelineName, ns)
	if err != nil {
		return nil, err
	}

	for _, s := range stages {
		if s.Spec.Order == currentOrder-1 {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("couldn't find stage for %v cd pipeline with %v order", pipelineName, currentOrder)
}

func findStagesRelatedToPipeline(client k8sClient.Client, cdPipeName, namespace string) ([]v1alpha1.Stage, error) {
	var stages v1alpha1.StageList
	if err := client.List(context.TODO(), &stages, &k8sClient.ListOptions{
		Namespace: namespace,
	}); err != nil {
		return nil, errors.Wrap(err, "couldn't list cd stages")
	}

	var res []v1alpha1.Stage
	for _, v := range stages.Items {
		if v.Spec.CdPipeline == cdPipeName {
			res = append(res, v)
		}
	}

	if res == nil {
		return nil, fmt.Errorf("no one stage were found by cd pipeline name %v", cdPipeName)
	}

	return res, nil
}
