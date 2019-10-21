package openshift

import (
	"fmt"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform/kubernetes"
	projectV1 "github.com/openshift/api/project/v1"
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"log"
)

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
		log.Fatal(err)
	}

	return &OpenshiftService{
		*client,
		projectClient}, nil
}

// Creates project in Openshift
func (service OpenshiftService) CreateProject(projectName string, projectDescription string) error {
	_, err := service.projectClient.ProjectRequests().Create(
		&projectV1.ProjectRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: projectName,
			},
			Description: projectDescription,
		},
	)

	if err != nil {
		errorMsg := fmt.Sprint(err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}

	log.Printf("Project %v has been created in Openshift", projectName)
	return nil
}
