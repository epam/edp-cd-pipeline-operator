package cluster

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

const (
	watchNamespaceEnvVar   = "WATCH_NAMESPACE"
	debugModeEnvVar        = "DEBUG_MODE"
	inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

func GetCdPipeline(c client.Client, name, namespace string) (*cdPipeApi.CDPipeline, error) {
	nsn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	i := &cdPipeApi.CDPipeline{}
	if err := c.Get(context.TODO(), nsn, i); err != nil {
		return nil, fmt.Errorf("failed to get cd pipeline: %w", err)
	}

	return i, nil
}

func GetCodebaseImageStream(c client.Client, name, namespace string) (*codebaseApi.CodebaseImageStream, error) {
	i := &codebaseApi.CodebaseImageStream{}

	if err := c.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, i); err != nil {
		return nil, fmt.Errorf("failed to get codebase image stream: %w", err)
	}

	return i, nil
}

// TODO: import this label from codebase-operator.
const CodebaseImageStreamCodebaseBranchLabel = "app.edp.epam.com/cbis-codebasebranch"

func GetCodebaseImageStreamByCodebaseBaseBranchName(
	ctx context.Context,
	k8sCl client.Client,
	codebaseBranchName string,
	namespace string,
) (*codebaseApi.CodebaseImageStream, error) {
	var codebaseImageStreamList codebaseApi.CodebaseImageStreamList

	if err := k8sCl.List(
		ctx,
		&codebaseImageStreamList,
		client.InNamespace(namespace),
		client.MatchingLabels{
			CodebaseImageStreamCodebaseBranchLabel: codebaseBranchName,
		},
	); err != nil {
		return nil, fmt.Errorf("failed to get CodebaseImageStream by label: %w", err)
	}

	if len(codebaseImageStreamList.Items) == 0 {
		return nil, fmt.Errorf("CodebaseImageStream not found for CodebaseBranch %s", codebaseBranchName)
	}

	if len(codebaseImageStreamList.Items) > 1 {
		return nil, fmt.Errorf("multiple CodebaseImageStream found for CodebaseBranch %s", codebaseBranchName)
	}

	return &codebaseImageStreamList.Items[0], nil
}

// GetWatchNamespace returns the namespace the operator should be watching for changes.
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}

	return ns, nil
}

// GetDebugMode returns the debug mode value.
func GetDebugMode() (bool, error) {
	mode, found := os.LookupEnv(debugModeEnvVar)
	if !found {
		return false, nil
	}

	b, err := strconv.ParseBool(mode)
	if err != nil {
		return false, fmt.Errorf("failed to parse bool: %w", err)
	}

	return b, nil
}

// Check whether the operator is running in cluster or locally.
func RunningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)
	return !os.IsNotExist(err)
}
