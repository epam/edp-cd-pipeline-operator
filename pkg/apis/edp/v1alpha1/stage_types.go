package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StageSpec defines the desired state of Stage
// +k8s:openapi-gen=true
type StageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Name         string        `json:"name"`
	CdPipeline   string        `json:"cdPipeline"`
	Description  string        `json:"description"`
	TriggerType  string        `json:"triggerType"`
	Order        int           `json:"order"`
	QualityGates []QualityGate `json:"qualityGates"`
}

type QualityGate struct {
	QualityGateType string  `json:"qualityGateType" valid:"Required"`
	StepName        string  `json:"stepName" valid:"Required;Match(/^[A-z0-9-._]/)"`
	AutotestName    *string `json:"autotestName"`
	BranchName      *string `json:"branchName"`
}

// StageStatus defines the observed state of Stage
// +k8s:openapi-gen=true
type StageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	LastTimeUpdated time.Time `json:"last_time_updated"`
	Status          string    `json:"status"`
}

type Autotest struct {
	AutotestName string `json:"autotestName"`
	BranchName   string `json:"branchName"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Stage is the Schema for the stages API
// +k8s:openapi-gen=true
type Stage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StageSpec   `json:"spec,omitempty"`
	Status StageStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StageList contains a list of Stage
type StageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stage{}, &StageList{})
}
