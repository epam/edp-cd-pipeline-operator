package stage

import (
	"context"
	"fmt"
	"time"

	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsClient "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/jenkins"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/service/helper"
	codebaseClient "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsApi "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	rbacV1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
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

	err = s.setupJenkins(cr)
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

func getPathToRepository(strategy, name string, url *string) string {
	if strategy == "import" {
		return *url
	}
	return "/" + name
}

func (s CDStageService) createStageConfig(stage *edpv1alpha1.Stage) (map[string]string, error) {
	var strStage string
	for i, qg := range stage.Spec.QualityGates {
		if i >= 1 {
			strStage = fmt.Sprintf("%v,", strStage)
		}
		strStage = fmt.Sprintf("%v{\"name\":\"%v\", \"step_name\":\"%v\"}", strStage, qg.QualityGateType, qg.StepName)
	}
	source := stage.Spec.Source
	jpm := map[string]string{
		"PIPELINE_NAME":         stage.Spec.CdPipeline,
		"STAGE_NAME":            stage.Name,
		"QG_STAGES":             strStage,
		"GIT_SERVER_CR_VERSION": "v2",
		"SOURCE_TYPE":           source.Type,
	}

	if source.Type == "library" {
		library, err := s.setLibraryParams(*stage)
		if err == nil {
			jpm["LIBRARY_URL"] = library["url"]
			jpm["LIBRARY_BRANCH"] = library["branch"]
			jpm["GIT_CREDENTIALS_ID"] = library["credentials"]
			jpm["GIT_SERVER_CR_NAME"] = library["gitServerName"]
		} else {
			jpm["SOURCE_TYPE"] = "default"
		}
	}

	return jpm, nil
}

func (s CDStageService) setLibraryParams(stage edpv1alpha1.Stage) (map[string]string, error) {
	cb, err := s.getLibraryParams(stage.Spec.Source.Library.Name, stage.Namespace)
	if err != nil {
		log.Error(err, "couldn't retrieve parameters for pipeline's library, default source type will be used",
			"Library name", stage.Spec.Source.Library.Name)
		return nil, err
	}
	gs, err := s.getGitServerParams(cb.Spec.GitServer, stage.Namespace)
	if err != nil {
		log.Error(err, "couldn't retrieve parameters for git server, default source type will be used",
			"Git server", cb.Spec.GitServer)
		return nil, err
	}
	return map[string]string{
		"url": fmt.Sprintf("ssh://%v@%v:%v%v", gs.Spec.GitUser, gs.Spec.GitHost, gs.Spec.SshPort,
			getPathToRepository(string(cb.Spec.Strategy), stage.Spec.Source.Library.Name, cb.Spec.GitUrlPath)),
		"credentials":   gs.Spec.NameSshKeySecret,
		"branch":        stage.Spec.Source.Library.Branch,
		"gitServerName": cb.Spec.GitServer,
	}, nil
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

func (s CDStageService) setupJenkins(stage *edpv1alpha1.Stage) error {
	jen, err := GetJenkins(s.Client, stage.Namespace)
	if err != nil {
		return errors.Wrap(err, "error in getting Jenkins CR")
	}

	jenkinsUrl := GetJenkinsUrl(*jen, stage.Namespace)
	jenkinsToken, jenkinsUsername, err := helper.GetJenkinsCreds(s.Platform, s.Client, stage.Namespace)
	if err != nil {
		return err
	}

	jenkins, err := jenkinsClient.Init(jenkinsUrl, jenkinsUsername, jenkinsToken)
	if err != nil {
		return err
	}

	stageConfig, err := s.createStageConfig(stage)
	if err != nil {
		return err
	}

	jpn := fmt.Sprintf("job-provisions/job/cd/job/%v", stage.Spec.JobProvisioning)
	err = jenkins.TriggerJobProvision(jpn, stageConfig)
	if err != nil {
		s.setFailedFields(edpv1alpha1.JenkinsConfiguration, err.Error())
		return errors.Wrap(err, "error in triggering job provisioner")
	}

	jobStatus, err := jenkins.GetJobStatus(jpn, 10*time.Second, 50)
	if err != nil {
		s.setFailedFields(edpv1alpha1.JenkinsConfiguration, err.Error())
		return errors.Wrap(err, "error in getting job provisioner status")
	}
	if jobStatus == "blue" {
		stage.Status = edpv1alpha1.StageStatus{
			LastTimeUpdated: time.Now(),
			Username:        "system",
			Action:          edpv1alpha1.JenkinsConfiguration,
			Result:          edpv1alpha1.Success,
			Value:           "active",
		}
		log.Info("cd pipeline has been created", "name", stage.Name)
	} else {
		log.Info("failed to create cd pipeline", "name", stage.Name, "status", jobStatus)
		s.setFailedFields(edpv1alpha1.JenkinsConfiguration, "Release job was failed.")
		return errors.New(fmt.Sprintf("failed to create cd pipeline %v", stage.Name))
	}
	return nil
}
