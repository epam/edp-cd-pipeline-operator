package cluster

import (
	"context"
	"fmt"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

const (
	watchNamespaceEnvVar   = "WATCH_NAMESPACE"
	debugModeEnvVar        = "DEBUG_MODE"
	inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
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

func GetCodebaseImageStream(client client.Client, name, namespace string) (*codebaseApi.CodebaseImageStream, error) {
	re := strings.NewReplacer("/", "-", ".", "-")
	name = re.Replace(name)
	nsn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	i := &codebaseApi.CodebaseImageStream{}
	if err := client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

// GetDebugMode returns the debug mode value
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, err
	}
	return b, nil
}

// Check whether the operator is running in cluster or locally
func RunningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)
	return !os.IsNotExist(err)
}
