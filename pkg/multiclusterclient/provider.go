package multiclusterclient

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
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

	restConfig, err := secretToRestConfig(secret)
	if err != nil {
		return nil, err
	}

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
	secret := &corev1.Secret{}
	if err := c.internalClusterClient.Get(
		ctx,
		client.ObjectKey{
			Namespace: secretNamespace,
			Name:      clusterName,
		},
		secret,
	); err != nil {
		return nil, fmt.Errorf("failed to get cluster secret: %w", err)
	}

	return secret, nil
}

const k8sClientConfigQPS = 50

func secretToRestConfig(s *corev1.Secret) (*rest.Config, error) {
	if _, ok := s.Data["config"]; !ok {
		return nil, fmt.Errorf("no config data in the secret %s", s.Name)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(s.Data["config"])
	if err != nil {
		return nil, fmt.Errorf("failed to create rest config from cluster secret: %w", err)
	}

	config.QPS = k8sClientConfigQPS
	config.Burst = int(config.QPS * 2)

	return config, nil
}
