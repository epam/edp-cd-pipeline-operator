package stage

import (
	"bytes"
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsClient "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/jenkins"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform"
	ph "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform/helper"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/service/helper"
	"github.com/pkg/errors"
	rbacV1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"text/template"
	"time"
)

var log = logf.Log.WithName("cd_stage_service")

const (
	StatusInit       = "initialized"
	StatusFailed     = "failed"
	StatusFinished   = "created"
	StatusInProgress = "in progress"
)

type CDStageService struct {
	Resource *edpv1alpha1.Stage
	Client   client.Client
	Platform platform.PlatformService
}

func (s CDStageService) CreateStage() error {
	cr := s.Resource
	reqLog := log.WithValues("stage name", cr.Spec.Name, "cd pipeline name", cr.Spec.CdPipeline,
		"namespace", cr.Namespace)
	reqLog.Info("Start creating Stage...")

	if cr.Status.Status != StatusInit {
		reqLog.Info("Stage is not in init status. Skipped")
		return nil
	}
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
		return errors.Wrap(err, "error has been occurred in cd_stage status update")
	}
	d, err := s.Platform.GetConfigMapData(cr.Namespace, "edp-config")
	edpName := d["edp_name"]
	if err != nil {
		s.setFailedFields(edpv1alpha1.FetchingUserSettingsConfigMap, err.Error())
		return errors.Wrap(err, "failed to fetch user settings config map")
	}
	err = s.setupPlatform(edpName, cr.Spec.CdPipeline, cr.Spec.Name, cr.Namespace)
	if err != nil {
		s.setFailedFields(edpv1alpha1.PlatformProjectCreation, err.Error())
		return errors.Wrap(err, "failed to setup platform")
	}
	stageStatus.Action = edpv1alpha1.PlatformProjectCreation
	err = s.updateStatus(stageStatus)
	if err != nil {
		return errors.Wrap(err, "error has been occurred in cd_stage status update")
	}

	ps, err := s.Platform.CreateStageJSON(*cr)
	if err != nil {
		return err
	}

	err = s.setupJenkins(cr.Namespace, cr.Spec.Name, cr.Spec.CdPipeline, ps)
	if err != nil {
		s.setFailedFields(edpv1alpha1.CreateJenkinsPipeline, err.Error())
		return errors.Wrap(err, "failed to setup Jenkins")
	}
	cr.Status = edpv1alpha1.StageStatus{
		Status:          StatusFinished,
		Available:       true,
		LastTimeUpdated: time.Now(),
		Username:        "system",
		Action:          edpv1alpha1.SetupDeploymentTemplates,
		Result:          edpv1alpha1.Success,
		Value:           "active",
	}

	reqLog.Info("Stage has been created")
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

	log.Info("Status for CD Stage", "cd stage name", s.Resource.Name)
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
}

func createStageConfig(name string, ps string) (*string, error) {
	var cdPipelineBuffer bytes.Buffer

	jenkinsName := map[string]interface{}{
		"name":               name,
		"gitServerCrVersion": "v2",
		"isOpenshift":        ph.IsOpenshift(),
		"pipelineStages":     ps,
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

func (s CDStageService) createRoleBinding(edpName string, projectName string, namespace string) error {
	err := s.Platform.CreateRoleBinding(
		edpName,
		projectName,
		rbacV1.RoleRef{Name: "admin", APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole"},
		[]rbacV1.Subject{
			{Kind: "Group", Name: edpName + "-edp-super-admin"},
			{Kind: "Group", Name: edpName + "-edp-admin"},
			{Kind: "ServiceAccount", Name: "jenkins", Namespace: namespace},
			{Kind: "ServiceAccount", Name: "edp-admin-console", Namespace: namespace},
		},
	)
	if err != nil {
		return err
	}

	return s.Platform.CreateRoleBinding(
		edpName,
		projectName,
		rbacV1.RoleRef{Name: "view", APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole"},
		[]rbacV1.Subject{
			{Kind: "Group", Name: edpName + "-edp-view"},
		},
	)
}

func (s CDStageService) setupPlatform(edpName string, cdPipelineName string, stageName string, namespace string) error {
	projectName := edpName + "-" + cdPipelineName + "-" + stageName

	err := s.Platform.CreateProject(projectName, "Deploy project for stage "+stageName)
	if err != nil {
		return err
	}

	return s.createRoleBinding(edpName, projectName, namespace)
}

func (s CDStageService) setupJenkins(namespace string, stageName string, cdPipelineName string, ps string) error {
	pipelineFolderName := cdPipelineName + "-cd-pipeline"
	jenkinsUrl := fmt.Sprintf("http://jenkins.%s:8080", namespace)
	jenkinsToken, jenkinsUsername, err := helper.GetJenkinsCreds(s.Platform, s.Client, namespace)
	if err != nil {
		return err
	}

	jenkins, err := jenkinsClient.Init(jenkinsUrl, jenkinsUsername, jenkinsToken)
	if err != nil {
		return err
	}

	stageConfig, err := createStageConfig(stageName, ps)
	if err != nil {
		return err
	}

	return jenkins.CreateJob(stageName, pipelineFolderName, *stageConfig)
}
