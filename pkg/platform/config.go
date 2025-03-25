package platform

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KrciConfigMap                    = "krci-config"
	KrciConfigContainerRegistryHost  = "container_registry_host"
	KrciConfigContainerRegistrySpace = "container_registry_space"
)

type KrciConfig struct {
	KrciConfigContainerRegistryHost  string
	KrciConfigContainerRegistrySpace string
}

func (c KrciConfig) GetContainerRegistryUrl() string {
	return fmt.Sprintf("%s/%s", c.KrciConfigContainerRegistryHost, c.KrciConfigContainerRegistrySpace)
}

func GetKrciConfig(ctx context.Context, k8sClient client.Client, namespace string) (*KrciConfig, error) {
	config := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      KrciConfigMap,
	}, config); err != nil {
		// backward compatibility for edp-config config map.
		if errors.IsNotFound(err) {
			return getEdpConfig(ctx, k8sClient, namespace)
		}

		return nil, fmt.Errorf("failed to get %s: %w", KrciConfigMap, err)
	}

	return mapConfigToKrciConfig(config)
}

// getEdpConfig is backward compatibility for edp-config config map.
// Deprecated: use GetKrciConfig instead.
// TODO: remove this function after all instances will be migrated to krci-config.
func getEdpConfig(ctx context.Context, k8sClient client.Client, namespace string) (*KrciConfig, error) {
	config := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      "edp-config",
	}, config); err != nil {
		return nil, fmt.Errorf("failed to get edp-config: %w", err)
	}

	return mapConfigToKrciConfig(config)
}

func mapConfigToKrciConfig(config *corev1.ConfigMap) (*KrciConfig, error) {
	c := &KrciConfig{
		KrciConfigContainerRegistryHost:  config.Data[KrciConfigContainerRegistryHost],
		KrciConfigContainerRegistrySpace: config.Data[KrciConfigContainerRegistrySpace],
	}

	if c.KrciConfigContainerRegistryHost == "" {
		return nil, fmt.Errorf("registry host is not set in %s config map", KrciConfigMap)
	}

	if c.KrciConfigContainerRegistrySpace == "" {
		return nil, fmt.Errorf("registry space is not set in %s config map", KrciConfigMap)
	}

	return c, nil
}
