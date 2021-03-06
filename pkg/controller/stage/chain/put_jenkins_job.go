package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/common"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	jenv1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type PutJenkinsJob struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

type qualityGate struct {
	Name     string `json:"name"`
	StepName string `json:"step_name"`
}

const (
	autoDeployTriggerType    = "Auto"
	qualityGateAutotestType  = "autotests"
	defaultAutoTriggerPeriod = 60
	apiVersion               = "v2.edp.epam.com/v1alpha1"
	jenkinsJobKind           = "JenkinsJob"
)

func (h PutJenkinsJob) ServeRequest(stage *v1alpha1.Stage) error {
	log := h.log.WithValues("stage name", stage.Name)
	log.Info("start creating jenkins job cr.")
	if err := h.tryToUpdateJenkinsJobConfig(*stage); err != nil {
		return errors.Wrapf(err, "failed to update %v JenkinsJob CR config", stage.Name)
	}
	if err := h.tryToCreateJenkinsJob(*stage); err != nil {
		return errors.Wrapf(err, "failed to create %v JenkinsJob CR", stage.Name)
	}
	log.Info("jenkins job cr has been created")
	return nextServeOrNil(h.next, stage)
}

func (h PutJenkinsJob) tryToCreateJenkinsJob(stage v1alpha1.Stage) error {
	h.log.Info("start creating JenkinsJob CR", "name", stage.Name)

	jc, err := h.createJenkinsJobConfig(stage)
	if err != nil {
		return err
	}

	jj := &jenv1alpha1.JenkinsJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       jenkinsJobKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      stage.Name,
			Namespace: stage.Namespace,
		},
		Spec: jenv1alpha1.JenkinsJobSpec{
			StageName:     &stage.Name,
			JenkinsFolder: &stage.Spec.CdPipeline,
			Job: jenv1alpha1.Job{
				Name:              fmt.Sprintf("job-provisions/job/cd/job/%v", stage.Spec.JobProvisioning),
				Config:            string(jc),
				AutoTriggerPeriod: common.GetInt32P(defaultAutoTriggerPeriod),
			},
		},
		Status: jenv1alpha1.JenkinsJobStatus{
			Action: v1alpha1.AcceptJenkinsJob,
		},
	}
	if err := h.client.Create(context.TODO(), jj); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			h.log.Info("jenkins job already exists. skip creating...", "name", stage.Name)
			return nil
		}
		return errors.Wrapf(err, "couldn't create jenkins job %v", jj.Name)
	}
	h.log.Info("JenkinsJob has been created", "name", stage.Name)
	return nil
}

func (h PutJenkinsJob) tryToUpdateJenkinsJobConfig(stage v1alpha1.Stage) error {
	jenkinsJob, err := h.getJenkinsJob(stage.Name, stage.Namespace)
	if k8serrors.IsNotFound(err) {
		h.log.Info("jenkins job does not exists. skip updating...", "name", stage.Name)
		return nil
	}
	if err != nil {
		return err
	}
	jc, err := h.createJenkinsJobConfig(stage)
	if err != nil {
		return err
	}

	jenkinsJob.Spec.Job.Config = string(jc)
	if err := h.client.Update(context.TODO(), jenkinsJob); err != nil {
		return err
	}
	h.log.Info("JenkinsJob config has been updated...", "name", stage.Name)
	return nil
}

func (h PutJenkinsJob) createJenkinsJobConfig(stage v1alpha1.Stage) ([]byte, error) {
	qgStages, err := getQualityGateStages(stage.Spec.QualityGates)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse quality gate stages")
	}

	dt, err := h.getDeploymentType(&stage)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get deploymentType value")
	}

	jpm := map[string]string{
		"PIPELINE_NAME":         stage.Spec.CdPipeline,
		"STAGE_NAME":            stage.Spec.Name,
		"QG_STAGES":             *qgStages,
		"GIT_SERVER_CR_VERSION": "v2",
		"SOURCE_TYPE":           stage.Spec.Source.Type,
		"AUTODEPLOY":            getAutoDeployStatus(stage.Spec.TriggerType),
		"DEPLOYMENT_TYPE":       *dt,
	}

	if stage.Spec.Source.Type == "library" {
		library, err := h.setLibraryParams(stage)
		if err == nil {
			jpm["LIBRARY_URL"] = library["url"]
			jpm["LIBRARY_BRANCH"] = library["branch"]
			jpm["GIT_CREDENTIALS_ID"] = library["credentials"]
			jpm["GIT_SERVER_CR_NAME"] = library["gitServerName"]
		} else {
			jpm["SOURCE_TYPE"] = "default"
		}
	}

	jc, err := json.Marshal(jpm)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't marshal parameters %v into json string", jpm)
	}
	return jc, nil
}

func (h PutJenkinsJob) getDeploymentType(stage *v1alpha1.Stage) (*string, error) {
	p, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return nil, err
	}
	return common.GetStringP(p.Spec.DeploymentType), nil
}

func getQualityGateStages(qualityGates []v1alpha1.QualityGate) (*string, error) {
	if qualityGates == nil || len(qualityGates) == 0 {
		return nil, nil
	}

	var stages []interface{}
	isPreviousStageAutotest := false
	for _, qg := range qualityGates {
		if qg.QualityGateType == qualityGateAutotestType {
			handleAutotestStage(qg, isPreviousStageAutotest, &stages)
			isPreviousStageAutotest = true
			continue
		}
		handleManualStage(qg, &stages)
		isPreviousStageAutotest = false
	}
	return getStagesInJson(stages)
}

func getStagesInJson(stages []interface{}) (*string, error) {
	jsonStages, err := json.Marshal(stages)
	if err != nil {
		return nil, err
	}
	return common.GetStringP(modifyQualityGateStagesJson(string(jsonStages))), nil
}

func modifyQualityGateStagesJson(qgStages string) string {
	qgStages = strings.TrimPrefix(qgStages, "[")
	qgStages = strings.TrimSuffix(qgStages, "]")
	return qgStages
}

func handleAutotestStage(qg v1alpha1.QualityGate, isPreviousStageAutotest bool, result *[]interface{}) {
	if isPreviousStageAutotest {
		handlePreviousAutotestStage(qg, result)
		return
	}
	*result = append(*result, qualityGate{
		Name:     qg.QualityGateType,
		StepName: qg.StepName,
	})
}

func handlePreviousAutotestStage(qg v1alpha1.QualityGate, result *[]interface{}) {
	switch old := (*result)[len(*result)-1].(type) {
	case []qualityGate:
		(*result)[len(*result)-1] = append((*result)[len(*result)-1].([]qualityGate), qualityGate{
			Name:     qg.QualityGateType,
			StepName: qg.StepName,
		})
	case qualityGate:
		(*result)[len(*result)-1] = []qualityGate{old, {
			Name:     qg.QualityGateType,
			StepName: qg.StepName,
		}}
	}
}

func handleManualStage(qg v1alpha1.QualityGate, result *[]interface{}) {
	*result = append(*result, qualityGate{
		Name:     qg.QualityGateType,
		StepName: qg.StepName,
	})
}

func (h PutJenkinsJob) setLibraryParams(stage v1alpha1.Stage) (map[string]string, error) {
	cb, err := h.getLibraryParams(stage.Spec.Source.Library.Name, stage.Namespace)
	if err != nil {
		h.log.Error(err, "couldn't retrieve parameters for pipeline's library, default source type will be used",
			"Library name", stage.Spec.Source.Library.Name)
		return nil, err
	}
	gs, err := h.getGitServerParams(cb.Spec.GitServer, stage.Namespace)
	if err != nil {
		h.log.Error(err, "couldn't retrieve parameters for git server, default source type will be used",
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

func (h PutJenkinsJob) getLibraryParams(name, ns string) (*codebaseApi.Codebase, error) {
	nsn := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	i := &codebaseApi.Codebase{}
	if err := h.client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

func (h PutJenkinsJob) getGitServerParams(name string, ns string) (*codebaseApi.GitServer, error) {
	nsn := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	i := &codebaseApi.GitServer{}
	if err := h.client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

func getPathToRepository(strategy, name string, url *string) string {
	if strategy == "import" {
		return *url
	}
	return "/" + name
}

func getAutoDeployStatus(tt string) string {
	if tt == autoDeployTriggerType {
		return "true"
	}
	return "false"
}

func (h PutJenkinsJob) getJenkinsJob(name, namespace string) (*jenv1alpha1.JenkinsJob, error) {
	nsn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	jj := &jenv1alpha1.JenkinsJob{}
	if err := h.client.Get(context.TODO(), nsn, jj); err != nil {
		return nil, err
	}
	return jj, nil
}
