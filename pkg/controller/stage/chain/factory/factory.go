package factory

import (
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	putCodebaseImageStream "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/put_codebase_image_stream"
	envLabel "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/put_environment_label_to_codebase_image_streams"
	putJenkinsJob "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/put_jenkins_job"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("cd_stage_factory")

const autoDeployTriggerType = "Auto"

func CreateDefChain(client client.Client, triggerType string) handler.CdStageHandler {
	return getChain(client, triggerType)
}

func getChain(client client.Client, triggerType string) handler.CdStageHandler {
	if autoDeployTriggerType == triggerType {
		log.Info("chain is selected", "type", "auto deploy")
		return putCodebaseImageStream.PutCodebaseImageStream{
			Next: putJenkinsJob.PutJenkinsJob{
				Client: client,
				Next: envLabel.PutEnvironmentLabelToCodebaseImageStreams{
					Client: client,
				},
			},
			Client: client,
		}
	}
	log.Info("chain is selected", "type", "manual deploy")
	return putCodebaseImageStream.PutCodebaseImageStream{
		Next: putJenkinsJob.PutJenkinsJob{
			Client: client,
		},
		Client: client,
	}
}
