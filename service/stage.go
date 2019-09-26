package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsClient "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/jenkins"
	Openshift "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/openshift"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/settings"
	rbacV1 "k8s.io/api/rbac/v1"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"
	"time"
)

type CDStageService struct {
	Resource *edpv1alpha1.Stage
	Client   client.Client
}

func (s CDStageService) CreateStage() error {
	cr := s.Resource

	log.Printf("Start creating Stage %v for CD Pipeline %v", cr.Spec.Name, cr.Spec.CdPipeline)
	if cr.Status.Status != StatusInit {
		log.Printf("Stage %v is not in init status. Skipped", cr.Spec.Name)
		return errors.New(fmt.Sprintf("Stage %v is not in init status. Skipped", cr.Spec.Name))
	}
	log.Printf("Stage %v has 'init' status", cr.Spec.Name)

	stageStatus := edpv1alpha1.StageStatus{
		Status:          StatusInProgress,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Result:          edpv1alpha1.Success,
		Username:        "system",
		Value:           "inactive",
	}

	stageStatus.Action = edpv1alpha1.AcceptCDStageRegistration
	err := s.updateStatus(stageStatus)
	if err != nil {
		return fmt.Errorf("error has been occurred in cd_stage status update: %v", err)
	}

	clientSet := Openshift.CreateOpenshiftClients()

	edpName, err := settings.GetUserSettingConfigMap(clientSet, cr.Namespace, "edp_name")
	if err != nil {
		log.Println("Couldn't fetch user settings config map")
		s.setFailedFields(edpv1alpha1.FetchingUserSettingsConfigMap, err.Error())
		return err
	}

	err = setupOpenshift(clientSet, edpName, cr.Spec.CdPipeline, cr.Spec.Name)
	if err != nil {
		log.Println("Couldn't setup Openshift client")
		s.setFailedFields(edpv1alpha1.OpenshiftProjectCreation, err.Error())
		return err
	}

	stageStatus.Action = edpv1alpha1.OpenshiftProjectCreation
	err = s.updateStatus(stageStatus)
	if err != nil {
		return fmt.Errorf("error has been occurred in cd_stage status update: %v", err)
	}

	err = setupJenkins(clientSet, s.Client, cr.Namespace, cr.Spec.Name, cr.Spec.CdPipeline)
	if err != nil {
		log.Println("Couldn't setup Jenkins")
		s.setFailedFields(edpv1alpha1.CreateJenkinsPipeline, err.Error())
		return err
	}

	stageStatus.Action = edpv1alpha1.CreateJenkinsPipeline
	err = s.updateStatus(stageStatus)
	if err != nil {
		return fmt.Errorf("error has been occurred in cd_stage status update: %v", err)
	}

	err = s.updateStatus(edpv1alpha1.StageStatus{
		Status:          StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.SetupDeploymentTemplates,
		Result:          edpv1alpha1.Success,
		Value:           "active",
	})
	if err != nil {
		return fmt.Errorf("error has been occurred in cd_stage status update: %v", err)
	}

	log.Printf("Stage %v has been created. Status: %v", cr.Name, StatusFinished)
	return nil
}

func (s CDStageService) updateStatus(status edpv1alpha1.StageStatus) error {
	s.Resource.Status = status

	err := s.Client.Status().Update(context.TODO(), s.Resource)
	if err != nil {
		err := s.Client.Update(context.TODO(), s.Resource)
		if err != nil {
			return err
		}
	}

	log.Printf("Status for CD Stage %v is set up.", s.Resource.Name)

	return nil
}

func (s CDStageService) setFailedFields(action edpv1alpha1.ActionType, message string) {
	s.Resource.Status = edpv1alpha1.StageStatus{
		Status:          StatusFailed,
		Available:       false,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          action,
		Result:          edpv1alpha1.Error,
		DetailedMessage: message,
		Value:           "failed",
	}

	log.Printf("Status %v for CD Stage %v is set up.", edpv1alpha1.Error, s.Resource.Name)
}

func createStageConfig(name string) (*string, error) {
	var cdPipelineBuffer bytes.Buffer

	jenkinsName := map[string]interface{}{
		"name":               name,
		"gitServerCrVersion": "v2",
	}

	tmpl, err := template.New("cd-pipeline.tmpl").ParseFiles("/usr/local/bin/pipelines/cd-pipeline.tmpl")
	if err != nil {
		return nil, err
	}

	if err := tmpl.Execute(&cdPipelineBuffer, jenkinsName); err != nil {
		return nil, err
	}

	cdPipeline := cdPipelineBuffer.String()

	return &cdPipeline, nil
}

func createRoleBinding(clientSet *Openshift.ClientSet, edpName string, projectName string) error {
	err := Openshift.CreateRoleBinding(
		clientSet,
		edpName,
		projectName,
		rbacV1.RoleRef{Name: "admin", APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole"},
		[]rbacV1.Subject{
			{Kind: "Group", Name: edpName + "-edp-super-admin"},
			{Kind: "Group", Name: edpName + "-edp-admin"},
			{Kind: "ServiceAccount", Name: "jenkins", Namespace: edpName + "-edp-cicd"},
			{Kind: "ServiceAccount", Name: "edp-admin-console", Namespace: edpName + "-edp-cicd"},
		},
	)
	if err != nil {
		return err
	}

	err = Openshift.CreateRoleBinding(
		clientSet,
		edpName,
		projectName,
		rbacV1.RoleRef{Name: "view", APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole"},
		[]rbacV1.Subject{
			{Kind: "Group", Name: edpName + "-edp-view"},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func setupOpenshift(clientSet *Openshift.ClientSet, edpName string, cdPipelineName string, stageName string) error {
	projectName := edpName + "-" + cdPipelineName + "-" + stageName

	err := Openshift.CreateProject(clientSet, projectName, "Deploy project for stage "+stageName)
	if err != nil {
		return err
	}

	err = createRoleBinding(clientSet, edpName, projectName)
	if err != nil {
		return err
	}
	return nil
}

func setupJenkins(clientSet *Openshift.ClientSet, k8sClient client.Client, namespace string, stageName string, cdPipelineName string) error {
	pipelineFolderName := cdPipelineName + "-cd-pipeline"
	jenkinsUrl := fmt.Sprintf("http://jenkins.%s:8080", namespace)
	jenkinsToken, jenkinsUsername, err := getJenkinsCreds(clientSet, k8sClient, namespace)
	if err != nil {
		return err
	}

	jenkins, err := jenkinsClient.Init(jenkinsUrl, jenkinsUsername, jenkinsToken)
	if err != nil {
		return err
	}

	stageConfig, err := createStageConfig(stageName)
	if err != nil {
		return err
	}

	err = jenkins.CreateJob(stageName, pipelineFolderName, *stageConfig)
	if err != nil {
		return err
	}
	return nil
}
