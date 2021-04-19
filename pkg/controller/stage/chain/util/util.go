package util

import (
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCdPipeline(client client.Client, stage *v1alpha1.Stage) (*v1alpha1.CDPipeline, error) {
	ownerPipe := helper.GetOwnerReference(consts.CDPipelineKind, stage.GetOwnerReferences())
	if ownerPipe != nil {
		return cluster.GetCdPipeline(client, ownerPipe.Name, stage.Namespace)
	}
	return cluster.GetCdPipeline(client, stage.Spec.CdPipeline, stage.Namespace)
}
