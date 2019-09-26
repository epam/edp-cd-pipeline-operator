package service

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsClient "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/jenkins"
	ClientSet "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/openshift"
	jenkinsApi "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsOperatorSpec "github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type CDPipelineService struct {
	Resource *edpv1alpha1.CDPipeline
	Client   client.Client
}

const (
	StatusInit       = "initialized"
	StatusFailed     = "failed"
	StatusFinished   = "created"
	StatusInProgress = "in progress"
)

func (s CDPipelineService) CreateCDPipeline() error {
	cr := s.Resource

	log.Printf("Start creating CD Pipeline: %v", cr.Spec.Name)
	if cr.Status.Status != StatusInit {
		log.Printf("CD Pipeline %v is not in init status. Skipped", cr.Spec.Name)
		return errors.New(fmt.Sprintf("CD Pipeline %v is not in init status. Skipped", cr.Spec.Name))
	}
	log.Printf("CD Pipeline %v has 'init' status", cr.Spec.Name)

	pipelineStatus := edpv1alpha1.CDPipelineStatus{
		Status:          StatusInProgress,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Result:          edpv1alpha1.Success,
		Username:        "system",
		Value:           "inactive",
	}

	pipelineStatus.Action = edpv1alpha1.AcceptCDPipelineRegistration
	err := s.updateStatus(pipelineStatus)
	if err != nil {
		return fmt.Errorf("error has been occurred in cd_pipeline status update: %v", err)
	}

	jenkinsUrl := fmt.Sprintf("http://jenkins.%s:8080", cr.Namespace)
	log.Printf("Jenkins URL has been generated: %v", jenkinsUrl)

	jenkinsToken, jenkinsUsername, err := getJenkinsCreds(ClientSet.CreateOpenshiftClients(), s.Client, cr.Namespace)
	if err != nil {
		log.Println("Couldn't fetch Jenkins creds")
		s.setFailedFields(edpv1alpha1.JenkinsConfiguration, err.Error())
		return err
	}

	jenkins, err := jenkinsClient.Init(jenkinsUrl, jenkinsUsername, jenkinsToken)
	if err != nil {
		log.Println("Couldn't initialize Jenkins client")
		s.setFailedFields(edpv1alpha1.JenkinsConfiguration, err.Error())
		return err
	}

	_, err = jenkins.CreateFolder(cr.Name + "-cd-pipeline")
	if err != nil {
		log.Println("Couldn't create folder for Jenkins")
		s.setFailedFields(edpv1alpha1.CreateJenkinsDirectory, err.Error())
		return err
	}

	pipelineStatus.Action = edpv1alpha1.CreateJenkinsDirectory
	err = s.updateStatus(pipelineStatus)
	if err != nil {
		return fmt.Errorf("error has been occurred in cd_pipeline status update: %v", err)
	}

	err = s.updateStatus(edpv1alpha1.CDPipelineStatus{
		Status:          StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.SetupInitialStructureForCDPipeline,
		Result:          edpv1alpha1.Success,
		Value:           "active",
	})
	if err != nil {
		return fmt.Errorf("error has been occurred in cd_pipeline status update: %v", err)
	}

	log.Printf("CD pipeline has been created. Status: %v", StatusFinished)
	return nil
}

func (s CDPipelineService) updateStatus(status edpv1alpha1.CDPipelineStatus) error {
	s.Resource.Status = status

	err := s.Client.Status().Update(context.TODO(), s.Resource)
	if err != nil {
		err := s.Client.Update(context.TODO(), s.Resource)
		if err != nil {
			return err
		}
	}

	log.Printf("Status for CD Pipeline %v is set up.", s.Resource.Name)

	return nil
}

func (s CDPipelineService) setFailedFields(action edpv1alpha1.ActionType, message string) {
	s.Resource.Status = edpv1alpha1.CDPipelineStatus{
		Status:          StatusFailed,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          action,
		Result:          edpv1alpha1.Error,
		DetailedMessage: message,
		Value:           "failed",
	}

	log.Printf("Status %v for CD Pipeline %v is set up.", edpv1alpha1.Error, s.Resource.Name)
}

func getJenkinsCreds(clientSet *ClientSet.ClientSet, k8sClient client.Client, namespace string) (string, string, error) {
	options := client.ListOptions{Namespace: namespace}
	jenkinsList := &jenkinsApi.JenkinsList{}

	err := k8sClient.List(context.TODO(), &options, jenkinsList)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return "", "", errors.Wrapf(err, "Jenkins installation is not found in namespace %v", namespace)
		}
		return "", "", errors.Wrapf(err, "Unable to get Jenkins CRs in namespace %v", namespace)
	}

	if len(jenkinsList.Items) == 0 {
		errors.Wrapf(err, "Jenkins installation is not found in namespace %v", namespace)
	}

	jenkins := &jenkinsList.Items[0]
	annotationKey := fmt.Sprintf("%v/%v", jenkinsOperatorSpec.EdpAnnotationsPrefix, jenkinsOperatorSpec.JenkinsTokenAnnotationSuffix)
	jenkinsTokenSecretName := jenkins.Annotations[annotationKey]
	jenkinsTokenSecret, err := clientSet.CoreClient.Secrets(namespace).Get(jenkinsTokenSecretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return "", "", errors.Wrapf(err, "Secret %v in not found", jenkinsTokenSecretName)
		}
		return "", "", errors.Wrapf(err, "Getting secret %v failed", jenkinsTokenSecretName)
	}
	return string(jenkinsTokenSecret.Data["password"]), string(jenkinsTokenSecret.Data["username"]), nil
}
