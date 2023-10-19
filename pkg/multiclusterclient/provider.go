package multiclusterclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

const (
	// nolint:gosec // it is not hardcoded credentials.
	argocdClusterSecretLabel    = "argocd.argoproj.io/secret-type"
	argocdClusterSecretLabelVal = "cluster"
)

type ClientProvider struct {
	internalClusterClient client.Client
}

// NewClientProvider creates a new ClientProvider instance.
func NewClientProvider(internalClusterClient client.Client) *ClientProvider {
	return &ClientProvider{internalClusterClient: internalClusterClient}
}

func (c *ClientProvider) GetClusterClient(ctx context.Context, secretNamespace, clusterName string, options client.Options) (client.Client, error) {
	if clusterName == "" || clusterName == cdPipeApi.InCluster {
		return c.internalClusterClient, nil
	}

	secret, err := c.getClusterSecret(ctx, clusterName, secretNamespace)
	if err != nil {
		return nil, err
	}

	cluster, err := secretToCluster(secret)
	if err != nil {
		return nil, err
	}

	restConfig := cluster.RestConfig()

	if options.Scheme == nil {
		options.Scheme = c.internalClusterClient.Scheme()
	}

	cl, err := client.New(restConfig, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return cl, nil
}

func (c *ClientProvider) getClusterSecret(ctx context.Context, clusterName, secretNamespace string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := c.internalClusterClient.List(
		ctx,
		secretList,
		client.InNamespace(secretNamespace),
		client.MatchingLabels(map[string]string{argocdClusterSecretLabel: argocdClusterSecretLabelVal}),
	); err != nil {
		return nil, fmt.Errorf("failed to get cluster secret %s: %w", clusterName, err)
	}

	for i := 0; i < len(secretList.Items); i++ {
		if string(secretList.Items[i].Data["name"]) == clusterName {
			return &secretList.Items[i], nil
		}
	}

	return nil, fmt.Errorf("secret for %s cluster not found", clusterName)
}

func secretToCluster(s *corev1.Secret) (*Cluster, error) {
	var config ClusterConfig
	if len(s.Data["config"]) > 0 {
		err := json.Unmarshal(s.Data["config"], &config)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal cluster config: %w", err)
		}
	}

	cluster := Cluster{
		Server: strings.TrimRight(string(s.Data["server"]), "/"),
		Name:   string(s.Data["name"]),
		Config: config,
	}

	return &cluster, nil
}
