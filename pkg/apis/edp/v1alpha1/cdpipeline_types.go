package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CDPipelineSpec defines the desired state of CDPipeline
// +k8s:openapi-gen=true
type CDPipelineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Name                  string   `json:"name"`
	CodebaseBranch        []string `json:"codebase_branch"`
	InputDockerStreams    []string `json:"input_docker_streams"`
	ThirdPartyServices    []string `json:"services"`
	ApplicationsToPromote []string `json:"applicationsToPromote"`
}

type ActionType string
type Result string

const (
	AcceptCDPipelineRegistration       ActionType = "accept_cd_pipeline_registration"
	AcceptCDStageRegistration          ActionType = "accept_cd_stage_registration"
	JenkinsConfiguration               ActionType = "jenkins_configuration"
	FetchingUserSettingsConfigMap      ActionType = "fetching_user_settings_config_map"
	OpenshiftProjectCreation           ActionType = "openshift_project_creation"
	SetupInitialStructureForCDPipeline ActionType = "setup_initial_structure"
	SetupDeploymentTemplates           ActionType = "setup_deployment_templates"

	Success Result = "success"
	Error   Result = "error"
)

// CDPipelineStatus defines the observed state of CDPipeline
// +k8s:openapi-gen=true
type CDPipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Available       bool       `json:"available"`
	LastTimeUpdated time.Time  `json:"last_time_updated"`
	Status          string     `json:"status"`
	Username        string     `json:"username"`
	Action          ActionType `json:"action"`
	Result          Result     `json:"result"`
	DetailedMessage string     `json:"detailed_message"`
	Value           string     `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CDPipeline is the Schema for the cdpipelines API
// +k8s:openapi-gen=true
type CDPipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CDPipelineSpec   `json:"spec,omitempty"`
	Status CDPipelineStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CDPipelineList contains a list of CDPipeline
type CDPipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CDPipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CDPipeline{}, &CDPipelineList{})
}
