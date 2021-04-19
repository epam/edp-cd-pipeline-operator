// +build !ignore_autogenerated

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.CDPipeline":       schema_pkg_apis_edp_v1alpha1_CDPipeline(ref),
		"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.CDPipelineSpec":   schema_pkg_apis_edp_v1alpha1_CDPipelineSpec(ref),
		"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.CDPipelineStatus": schema_pkg_apis_edp_v1alpha1_CDPipelineStatus(ref),
		"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.Stage":            schema_pkg_apis_edp_v1alpha1_Stage(ref),
		"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.StageSpec":        schema_pkg_apis_edp_v1alpha1_StageSpec(ref),
		"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.StageStatus":      schema_pkg_apis_edp_v1alpha1_StageStatus(ref),
	}
}

func schema_pkg_apis_edp_v1alpha1_CDPipeline(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "CDPipeline is the Schema for the cdpipelines API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.CDPipelineSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.CDPipelineStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.CDPipelineSpec", "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.CDPipelineStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_edp_v1alpha1_CDPipelineSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "CDPipelineSpec defines the desired state of CDPipeline",
				Properties: map[string]spec.Schema{
					"name": {
						SchemaProps: spec.SchemaProps{
							Description: "INSERT ADDITIONAL SPEC FIELDS - desired state of cluster Important: Run \"operator-sdk generate k8s\" to regenerate code after modifying this file Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"codebaseBranch": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
				},
				Required: []string{"name", "codebaseBranch"},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_edp_v1alpha1_CDPipelineStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "CDPipelineStatus defines the observed state of CDPipeline",
				Properties: map[string]spec.Schema{
					"lastTimeUpdated": {
						SchemaProps: spec.SchemaProps{
							Description: "INSERT ADDITIONAL STATUS FIELD - define observed state of cluster Important: Run \"operator-sdk generate k8s\" to regenerate code after modifying this file Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html",
							Type:        []string{"string"},
							Format:      "date-time",
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
				Required: []string{"lastTimeUpdated", "status"},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_edp_v1alpha1_Stage(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Stage is the Schema for the stages API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.StageSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.StageStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.StageSpec", "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1.StageStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_edp_v1alpha1_StageSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "StageSpec defines the desired state of Stage",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_edp_v1alpha1_StageStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "StageStatus defines the observed state of Stage",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}
