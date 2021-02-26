package put_jenkins_job

import (
	"context"
	"encoding/json"
	"fmt"
	codebasev1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	jenv1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type PutJenkinsJob struct {
	Next   handler.CdStageHandler
	Client client.Client
}

const autoDeployTriggerType = "Auto"

var log = logf.Log.WithName("put_jenkins_job_chain")

func (h PutJenkinsJob) ServeRequest(stage *v1alpha1.Stage) error {
	vLog := log.WithValues("stage name", stage.Name)
	vLog.Info("start creating jenkins job cr.")
	if err := h.tryToCreateJenkinsJob(*stage); err != nil {
		return errors.Wrapf(err, "failed to create %v JenkinsJob CR", stage.Name)
	}
	vLog.Info("jenkins job cr has been created")
	return handler.NextServeOrNil(h.Next, stage)
}

func (h PutJenkinsJob) tryToCreateJenkinsJob(stage v1alpha1.Stage) error {
	log.Info("start creating JenkinsJob CR", "name", stage.Name)

	jc, _ := h.createJenkinsJobConfig(stage)

	jj := &jenv1alpha1.JenkinsJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1alpha1",
			Kind:       "JenkinsJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      stage.Name,
			Namespace: stage.Namespace,
		},
		Spec: jenv1alpha1.JenkinsJobSpec{
			StageName:     &stage.Name,
			JenkinsFolder: &stage.Spec.CdPipeline,
			Job: jenv1alpha1.Job{
				Name:   fmt.Sprintf("job-provisions/job/cd/job/%v", stage.Spec.JobProvisioning),
				Config: string(jc),
			},
		},
		Status: jenv1alpha1.JenkinsJobStatus{
			Action: v1alpha1.AcceptJenkinsJob,
		},
	}
	if err := h.Client.Create(context.TODO(), jj); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Info("jenkins job already exists. skip creating...", "name", stage.Name)
			return nil
		}
		return errors.Wrapf(err, "couldn't create jenkins job %v", jj.Name)
	}
	log.Info("JenkinsJob has been created", "name", stage.Name)
	return nil
}

func (h PutJenkinsJob) createJenkinsJobConfig(stage v1alpha1.Stage) ([]byte, error) {
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
		"STAGE_NAME":            stage.Spec.Name,
		"QG_STAGES":             strStage,
		"GIT_SERVER_CR_VERSION": "v2",
		"SOURCE_TYPE":           source.Type,
	}

	if source.Type == "library" {
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

	if stage.Spec.TriggerType == autoDeployTriggerType {
		jpm["AUTODEPLOY"] = "true"
	}

	jc, err := json.Marshal(jpm)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't marshal parameters %v into json string", jpm)
	}
	return jc, nil
}

func (h PutJenkinsJob) setLibraryParams(stage v1alpha1.Stage) (map[string]string, error) {
	cb, err := h.getLibraryParams(stage.Spec.Source.Library.Name, stage.Namespace)
	if err != nil {
		log.Error(err, "couldn't retrieve parameters for pipeline's library, default source type will be used",
			"Library name", stage.Spec.Source.Library.Name)
		return nil, err
	}
	gs, err := h.getGitServerParams(cb.Spec.GitServer, stage.Namespace)
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

func (h PutJenkinsJob) getLibraryParams(name, ns string) (*codebasev1alpha1.Codebase, error) {
	nsn := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	i := &codebasev1alpha1.Codebase{}
	if err := h.Client.Get(context.TODO(), nsn, i); err != nil {
		return nil, err
	}
	return i, nil
}

func (h PutJenkinsJob) getGitServerParams(name string, ns string) (*codebasev1alpha1.GitServer, error) {
	nsn := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	i := &codebasev1alpha1.GitServer{}
	if err := h.Client.Get(context.TODO(), nsn, i); err != nil {
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
