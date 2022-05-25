package v1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CDPipelineSpec defines the desired state of CDPipeline
type CDPipelineSpec struct {
	// +kubebuilder:validation:MinLength=2

	// Name of CD pipeline
	Name string `json:"name"`

	// Which type of kind will be deployed e.g. Container, Custom
	DeploymentType string `json:"deploymentType"`

	// +kubebuilder:validation:MinItems=1

	// A list of docker streams
	InputDockerStreams []string `json:"inputDockerStreams"`

	// +kubebuilder:validation:MinItems=1

	// A list of applications included in CDPipeline.
	Applications []string `json:"applications"`

	// A list of applications which will promote after successful release.
	// +nullable
	// +optional
	ApplicationsToPromote []string `json:"applicationsToPromote,omitempty"`
}

type ActionType string

const (
	AcceptCDStageRegistration          ActionType = "accept_cd_stage_registration"
	SetupInitialStructureForCDPipeline ActionType = "setup_initial_structure"
	AcceptJenkinsJob                              = "accept_jenkins_job"
)

// Result describes how action were performed.
// Once action ended, we record a result of this action.
// +kubebuilder:validation:Enum=success;error
type Result string

const (
	// Success result of operation.
	Success Result = "success"

	// Error result point to unsuccessful operation.
	Error Result = "error"
)

// CDPipelineStatus defines the observed state of CDPipeline
type CDPipelineStatus struct {
	// This flag indicates neither CDPipeline are initialized and ready to work. Defaults to false.
	Available bool `json:"available"`

	// Information when  the last time the action were performed.
	LastTimeUpdated metaV1.Time `json:"last_time_updated"`

	// Specifies a current status of CDPipeline.
	Status string `json:"status"`

	// Name of user who made a last change.
	Username string `json:"username"`

	// The last Action was performed.
	Action ActionType `json:"action"`

	// A result of an action which were performed.
	// - "success": action where performed successfully;
	// - "error": error has occurred;
	Result Result `json:"result"`

	// Detailed information regarding action result
	// which were performed
	// +optional
	DetailedMessage string `json:"detailed_message,omitempty"`

	// Specifies a current state of CDPipeline.
	Value string `json:"value"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// CDPipeline is the Schema for the cdpipelines API
type CDPipeline struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CDPipelineSpec   `json:"spec,omitempty"`
	Status CDPipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CDPipelineList contains a list of CDPipeline
type CDPipelineList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []CDPipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CDPipeline{}, &CDPipelineList{})
}
