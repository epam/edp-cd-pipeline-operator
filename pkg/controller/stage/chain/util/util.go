package util

import (
	"fmt"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const previousStageNameAnnotationKey = "deploy.edp.epam.com/previous-stage-name"

func GetCdPipeline(client k8sClient.Client, stage *v1alpha1.Stage) (*v1alpha1.CDPipeline, error) {
	ownerPipe := helper.GetOwnerReference(consts.CDPipelineKind, stage.GetOwnerReferences())
	if ownerPipe != nil {
		return cluster.GetCdPipeline(client, ownerPipe.Name, stage.Namespace)
	}
	return cluster.GetCdPipeline(client, stage.Spec.CdPipeline, stage.Namespace)
}

func FindPreviousStageName(annotations map[string]string) (string, error) {
	if annotations == nil {
		return "", fmt.Errorf("there're no any annotation")
	}

	if val, ok := annotations[previousStageNameAnnotationKey]; ok {
		return val, nil
	}

	return "", fmt.Errorf("stage doesnt contain %v annotation", previousStageNameAnnotationKey)
}
