package stage

import (
	"bytes"
	"context"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsClient "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/jenkins"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/service/helper"
	codebaseClient "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/pkg/errors"
	rbacV1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
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

	pipeStg, err := s.Platform.CreateStageJSON(*cr)
	if err != nil {
		return err
	}

	err = s.setupJenkins(cr, pipeStg)
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

func (s CDStageService) getLibraryParams(libName string, ns string) (*codebaseClient.Codebase, error) {
	nsn := types.NamespacedName{
		Namespace: ns,
		Name:      libName,
	}

	cb := &codebaseClient.Codebase{}
	err := s.Client.Get(context.TODO(), nsn, cb)
	if err != nil {
		return nil, err
	}
	return cb, nil
}

func (s CDStageService) getGitServerParams(gsName string, ns string) (*codebaseClient.GitServer, error) {
	nsn := types.NamespacedName{
		Namespace: ns,
		Name:      gsName,
	}

	gs := &codebaseClient.GitServer{}
	err := s.Client.Get(context.TODO(), nsn, gs)
	if err != nil {
		return nil, err
	}
	return gs, nil
}

func (s CDStageService) getPipeSrcParams(stage *edpv1alpha1.Stage, pipeSrc map[string]interface{}) map[string]interface{} {
	if cb, err := s.getLibraryParams(stage.Spec.Source.Library.Name, stage.Namespace); err != nil {
		log.Error(err, "Couldn't retrieve parameters for pipeline's library, default source type will be used", "Library name", stage.Spec.Source.Library.Name)
	} else {
		if gs, err := s.getGitServerParams(cb.Spec.GitServer, stage.Namespace); err != nil {
			log.Error(err, "Couldn't retrieve parameters for git server, default source type will be used", "Git server", cb.Spec.GitServer)
		} else {
			pipeSrc["type"] = "library"
			pipeSrc["library"] = map[string]string{
				"url":         fmt.Sprintf("ssh://%v@%v:%v/%v", gs.Spec.GitUser, gs.Spec.GitHost, gs.Spec.SshPort, stage.Spec.Source.Library.Name),
				"credentials": gs.Spec.NameSshKeySecret,
				"branch":      stage.Spec.Source.Library.Branch,
			}
		}
	}
	return pipeSrc
}

func (s CDStageService) createStageConfig(stage *edpv1alpha1.Stage, ps string) (*string, error) {
	var cdPipelineBuffer bytes.Buffer

	pipeSrc := map[string]interface{}{
		"type":    "default",
		"library": map[string]string{},
	}

	if stage.Spec.Source.Type == "library" {
		pipeSrc = s.getPipeSrcParams(stage, pipeSrc)
	}

	pipelineStruct := map[string]interface{}{
		"name":               stage.Spec.Name,
		"gitServerCrVersion": "v2",
		"pipelineStages":     ps,
		"source":             pipeSrc,
	}

	tmpl, err := template.New("cd-pipeline.tmpl").ParseFiles("/usr/local/bin/pipelines/cd-pipeline.tmpl")
	if err != nil {
		return nil, err
	}

	if err := tmpl.Execute(&cdPipelineBuffer, pipelineStruct); err != nil {
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

func (s CDStageService) setupJenkins(stage *edpv1alpha1.Stage, pipeStg string) error {
	pipelineFolderName := stage.Spec.CdPipeline + "-cd-pipeline"
	jenkinsUrl := fmt.Sprintf("http://jenkins.%s:8080", stage.Namespace)
	jenkinsToken, jenkinsUsername, err := helper.GetJenkinsCreds(s.Platform, s.Client, stage.Namespace)
	if err != nil {
		return err
	}

	jenkins, err := jenkinsClient.Init(jenkinsUrl, jenkinsUsername, jenkinsToken)
	if err != nil {
		return err
	}

	stageConfig, err := s.createStageConfig(stage, pipeStg)
	if err != nil {
		return err
	}

	return jenkins.CreateJob(stage.Spec.Name, pipelineFolderName, *stageConfig)
}
