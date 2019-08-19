package service

import (
	"bytes"
	edpv1alpha1 "cd-pipeline-operator/pkg/apis/edp/v1alpha1"
	jenkinsClient "cd-pipeline-operator/pkg/jenkins"
	Openshift "cd-pipeline-operator/pkg/openshift"
	"cd-pipeline-operator/pkg/settings"
	"errors"
	"fmt"
	rbacV1 "k8s.io/api/rbac/v1"
	"log"
	"text/template"
	"time"
)

func CreateStage(cr *edpv1alpha1.Stage) error {
	log.Printf("Start creating Stage %v for CD Pipeline %v", cr.Spec.Name, cr.Spec.CdPipeline)
	if cr.Status.Status != StatusInit {
		log.Printf("Stage %v is not in init status. Skipped", cr.Spec.Name)
		return errors.New(fmt.Sprintf("Stage %v is not in init status. Skipped", cr.Spec.Name))
	}
	log.Printf("Stage %v has 'init' status", cr.Spec.Name)

	setStageStatusFields(cr, StatusInProgress, time.Now())

	clientSet := Openshift.CreateOpenshiftClients()
	edpName, err := settings.GetUserSettingConfigMap(clientSet, cr.Namespace, "edp_name")
	if err != nil {
		log.Println("Couldn't fetch user settings config map")
		rollbackStage(cr)
		return err
	}

	err = setupOpenshift(clientSet, edpName, cr.Spec.CdPipeline, cr.Spec.Name)
	if err != nil {
		log.Println("Couldn't setup Openshift client")
		rollbackStage(cr)
		return err
	}

	err = setupJenkins(clientSet, cr.Namespace, cr.Spec.Name, cr.Spec.CdPipeline)
	if err != nil {
		log.Println("Couldn't setup Jenkins")
		rollbackStage(cr)
		return err
	}

	setStageStatusFields(cr, StatusFinished, time.Now())
	log.Printf("Stage %v has been created. Status: %v", cr.Name, StatusFinished)
	return nil
}

func rollbackStage(cr *edpv1alpha1.Stage) {
	setStageStatusFields(cr, StatusFailed, time.Now())
}

func setStageStatusFields(cr *edpv1alpha1.Stage, status string, time time.Time) {
	cr.Status.Status = status
	cr.Status.LastTimeUpdated = time
	log.Printf("Status for stage %v has been updated to '%v' at %v.", cr.Spec.Name, status, time)
}

func createStageConfig(name string) (*string, error) {
	var cdPipelineBuffer bytes.Buffer

	jenkinsName := map[string]interface{}{
		"name": name,
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
			{Kind: "ServiceAccount", Name: "admin-console", Namespace: edpName + "-edp-cicd"},
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

func setupJenkins(clientSet *Openshift.ClientSet, namespace string, stageName string, cdPipelineName string) error {
	pipelineFolderName := cdPipelineName + "-cd-pipeline"
	jenkinsUrl := fmt.Sprintf("http://jenkins.%s:8080", namespace)
	jenkinsToken, jenkinsUsername, err := getJenkinsCreds(clientSet, namespace)
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
