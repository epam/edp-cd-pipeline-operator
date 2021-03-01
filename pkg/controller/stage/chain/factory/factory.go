package factory

import (
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/deleteenvironmentlabelfromcodebaseimagestreams"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/handler"
	putCodebaseImageStream "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/put_codebase_image_stream"
	envLabel "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/put_environment_label_to_codebase_image_streams"
	putJenkinsJob "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/controller/stage/chain/put_jenkins_job"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/util/consts"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("cd_stage_factory")

func CreateDefChain(client client.Client, triggerType string) handler.CdStageHandler {
	return getChain(client, triggerType)
}

func CreateDeleteChain(client client.Client) handler.CdStageHandler {
	log.Info("deletion chain is selected", "type", "auto deploy")
	return deleteenvironmentlabelfromcodebaseimagestreams.DeleteEnvironmentLabelFromCodebaseImageStreams{
		Client: client,
	}
}

func getChain(client client.Client, triggerType string) handler.CdStageHandler {
	if consts.AutoDeployTriggerType == triggerType {
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
