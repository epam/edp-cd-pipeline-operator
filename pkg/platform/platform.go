package platform

import (
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform/helper"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform/kubernetes"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform/openshift"
	"github.com/pkg/errors"
	rbacV1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

// PlatformService interface
type PlatformService interface {
	CreateProject(projectName string, projectDescription string) error
	CreateRoleBinding(edpName string, namespace string, roleRef rbacV1.RoleRef, subjects []rbacV1.Subject) error
	GetSecretData(namespace string, name string) (map[string][]byte, error)
	GetConfigMapData(namespace string, name string) (map[string]string, error)
}

// NewPlatformService returns platform service interface implementation
func NewPlatformService(p string) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get rest configs for platform")
	}

	switch strings.ToLower(p) {
	case helper.PlatformOpenshift:
		platform, err := openshift.New(restConfig)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to init for Openshift platform")
		}
		return platform, nil
	case helper.PlatformKubernetes:
		platform, err := kubernetes.New(restConfig)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to init for Kubernetes platform")
		}
		return platform, nil
	default:
		return nil, errors.New("Unknown platform type")
	}
}
