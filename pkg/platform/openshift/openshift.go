package openshift

import (
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform/kubernetes"
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jenkins")

// OpenshiftService struct for Openshift platform service
type OpenshiftService struct {
	kubernetes.K8SService

	projectClient *projectV1Client.ProjectV1Client
}

// New initializes OpenshiftService
func New(config *rest.Config) (*OpenshiftService, error) {
	client, err := kubernetes.New(config)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to init K8S platform service")
	}

	projectClient, err := projectV1Client.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to init project client for Openshift")
	}

	return &OpenshiftService{
		*client,
		projectClient}, nil
}
