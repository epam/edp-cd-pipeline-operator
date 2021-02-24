package cluster

import (
	"context"
	v1alphaCodebase "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCdPipeline(client client.Client, name, namespace string) (*v1alpha1.CDPipeline, error) {
	nsn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	i := &v1alpha1.CDPipeline{}
	if err := client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

func GetCodebaseImageStream(client client.Client, name, namespace string) (*v1alphaCodebase.CodebaseImageStream, error) {
	nsn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	i := &v1alphaCodebase.CodebaseImageStream{}
	if err := client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}
