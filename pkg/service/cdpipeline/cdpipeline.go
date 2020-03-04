package cdpipeline

import (
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsClient "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/jenkins"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/service/helper"
	jenkinsApi "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"time"
)

var log = logf.Log.WithName("cd_pipeline_service")

const (
	StatusInit       = "initialized"
	StatusFailed     = "failed"
	StatusFinished   = "created"
	StatusInProgress = "in progress"
)

type CDPipelineService struct {
	Resource *edpv1alpha1.CDPipeline
	Client   client.Client
	Platform platform.PlatformService
}

func GetJenkinsUrl(jenkins jenkinsApi.Jenkins, namespace string) string {
	basePath := ""
	if len(jenkins.Spec.BasePath) > 0 {
		basePath = fmt.Sprintf("/%v", jenkins.Spec.BasePath)
	}
	return fmt.Sprintf("http://jenkins.%s:8080%v", namespace, basePath)
}

func GetJenkins(k8sClient client.Client, namespace string) (*jenkinsApi.Jenkins, error) {
	options := client.ListOptions{Namespace: namespace}
	jenkinsList := &jenkinsApi.JenkinsList{}

	err := k8sClient.List(context.TODO(), &options, jenkinsList)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to get Jenkins CRs in namespace %v", namespace)
	}

	if len(jenkinsList.Items) == 0 {
		return nil, fmt.Errorf("jenkins installation is not found in namespace %v", namespace)
	}

	return &jenkinsList.Items[0], nil
}

func (s CDPipelineService) CreateCDPipeline() error {
	cr := s.Resource
	reqLog := log.WithValues("CD pipeline name", cr.Spec.Name, "namespace", cr.Namespace)
	reqLog.Info("Start creating CD Pipeline...")

	if cr.Status.Status != StatusInit {
		reqLog.Info("CD Pipeline is not in init status. Skipped")
		return nil
	}
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
		return errors.Wrap(err, "error has been occurred in cd_pipeline status update")
	}

	jen, err := GetJenkins(s.Client, cr.Namespace)
	if err != nil {
		return errors.Wrap(err, "error in getting Jenkins CR")
	}

	jenkinsUrl := GetJenkinsUrl(*jen, cr.Namespace)
	jenkinsToken, jenkinsUsername, err := helper.GetJenkinsCreds(s.Platform, s.Client, cr.Namespace)
	if err != nil {
		s.setFailedFields(edpv1alpha1.JenkinsConfiguration, err.Error())
		return errors.Wrap(err, "failed to get jenkins credentials")
	}
	jenkins, err := jenkinsClient.Init(jenkinsUrl, jenkinsUsername, jenkinsToken)
	if err != nil {
		s.setFailedFields(edpv1alpha1.JenkinsConfiguration, err.Error())
		return errors.Wrap(err, "failed to init Jenkins clients")
	}
	_, err = jenkins.CreateFolder(cr.Name + "-cd-pipeline")
	if err != nil {
		s.setFailedFields(edpv1alpha1.CreateJenkinsDirectory, err.Error())
		return errors.Wrap(err, "failed to create folder in Jenkins")
	}
	cr.Status = edpv1alpha1.CDPipelineStatus{
		Status:          StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.SetupInitialStructureForCDPipeline,
		Result:          edpv1alpha1.Success,
		Value:           "active",
	}

	reqLog.Info("CD pipeline has been created")
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
	log.Info("Status for CD Pipeline is set up", "cd pipeline name", s.Resource.Name)
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
}
