package cluster

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
)

const (
	watchNamespaceEnvVar   = "WATCH_NAMESPACE"
	debugModeEnvVar        = "DEBUG_MODE"
	inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

func GetCdPipeline(client client.Client, name, namespace string) (*cdPipeApi.CDPipeline, error) {
	nsn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	i := &cdPipeApi.CDPipeline{}
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
	if err := client.Get(context.Background(), nsn, i); err != nil {
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

// JenkinsEnabled returns true if jenkins is enabled in the namespace.
func JenkinsEnabled(ctx context.Context, k8sClient client.Reader, namespace string, log logr.Logger) bool {
	jenkinsList := &jenkinsApi.JenkinsList{}

	if err := k8sClient.List(ctx, jenkinsList, &client.ListOptions{Namespace: namespace}); err != nil {
		log.Error(err, "unable to get jenkins list")

		return false
	}

	return len(jenkinsList.Items) != 0
}
