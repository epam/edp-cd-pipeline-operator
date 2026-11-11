// Package security maps Trivy Operator reports of Stage namespaces to DefectDojo imports.
package security

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/defectdojo"
)

const (
	// TrivyGroup is the API group of Trivy Operator reports.
	TrivyGroup = "aquasecurity.github.io"
	// TrivyVersion is the API version of Trivy Operator reports.
	TrivyVersion = "v1alpha1"

	KindVulnerabilityReport = "VulnerabilityReport"
	KindExposedSecretReport = "ExposedSecretReport"

	LabelResourceKind = "trivy-operator.resource.kind"
	LabelResourceName = "trivy-operator.resource.name"

	LabelResourceNamespace = "trivy-operator.resource.namespace"
	LabelContainerName     = "trivy-operator.container.name"

	TagScopeRuntime    = "krci-scope:runtime"
	TagPrefixTenant    = "krci-tenant"
	TagPrefixEnv       = "krci-env"
	TagPrefixNamespace = "krci-namespace"

	// EngagementPrefixRuntime prefixes the runtime engagement name: "runtime:<cdPipeline>-<stage>".
	EngagementPrefixRuntime = "runtime"
	// ProductPrefixEnvironment prefixes the product of workloads that belong to no codebase: "env:<cdPipeline>-<stage>".
	ProductPrefixEnvironment = "env"

	tagPrefixWorkload  = "krci-workload"
	tagPrefixContainer = "krci-container"
	tagPrefixCodebase  = "krci-codebase"

	digestVersionLen = len("sha256:") + 12
)

// ReportKinds lists the Trivy Operator report kinds that are imported.
var ReportKinds = []string{KindVulnerabilityReport, KindExposedSecretReport}

// ReportGVK returns the GroupVersionKind of a report kind.
func ReportGVK(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: TrivyGroup, Version: TrivyVersion, Kind: kind}
}

// TenantTag is the tag that ties a DefectDojo test to the operator that wrote it.
func TenantTag(tenant string) string { return TagPrefixTenant + ":" + tenant }

// NamespaceTag is the tag that ties a DefectDojo test to the namespace of a Stage.
func NamespaceTag(namespace string) string { return TagPrefixNamespace + ":" + namespace }

// ImportTitle is the title of the DefectDojo test of a workload container:
// "<ReportKind> <namespace>/<kind>/<name>/<container>".
// It is unique per namespace, so tests of equally named workloads of two namespaces never meet.
func ImportTitle(reportKind, namespace string, workload Workload, container string) string {
	return reportKind + " " + namespace + "/" + workload.String() + "/" + container
}

// EnvTag is the tag that ties a DefectDojo test to a Stage.
func EnvTag(cdPipeline, stage string) string { return TagPrefixEnv + ":" + cdPipeline + "." + stage }

// Environment identifies a Stage namespace of a tenant.
type Environment struct {
	Tenant     string
	CDPipeline string
	Stage      string
	Namespace  string
}

// Import is the DefectDojo destination and the descriptive fields of one report.
type Import struct {
	Product    string
	Engagement string
	TestTitle  string
	Version    string
	Tags       []string
	// Codebase is empty for images that belong to no codebase of the tenant.
	Codebase string
}

// ReportImport derives the DefectDojo destination of a report of a workload container in a Stage namespace.
// The product is the codebase that owns the image, or an environment product for any other image.
func ReportImport(
	env Environment,
	report *unstructured.Unstructured,
	workload Workload,
	codebases map[string]string,
) Import {
	image := ImageOf(report)
	codebase := codebases[image.Name]
	container := report.GetLabels()[LabelContainerName]
	engagement := EngagementPrefixRuntime + ":" + env.CDPipeline + "-" + env.Stage

	product := ProductPrefixEnvironment + ":" + env.CDPipeline + "-" + env.Stage
	if codebase != "" {
		product = codebase
	}

	tags := []string{
		TagScopeRuntime,
		TenantTag(env.Tenant),
		EnvTag(env.CDPipeline, env.Stage),
		NamespaceTag(env.Namespace),
		tagPrefixWorkload + ":" + workload.Tag(),
		tagPrefixContainer + ":" + container,
	}
	if codebase != "" {
		tags = append(tags, tagPrefixCodebase+":"+codebase)
	}

	version := image.Tag
	if version == "" {
		version = image.Digest[:min(len(image.Digest), digestVersionLen)]
	}

	return Import{
		Product:    product,
		Engagement: engagement,
		TestTitle:  ImportTitle(report.GetKind(), env.Namespace, workload, container),
		Version:    version,
		Tags:       tags,
		Codebase:   codebase,
	}
}

// Request completes the import into a DefectDojo re-import request.
func (i Import) Request(productType string, scanDate time.Time, report []byte) defectdojo.ReimportRequest {
	return defectdojo.ReimportRequest{
		ProductType: productType,
		Product:     i.Product,
		Engagement:  i.Engagement,
		TestTitle:   i.TestTitle,
		Version:     i.Version,
		Tags:        i.Tags,
		ScanDate:    scanDate,
		Report:      report,
	}
}

// Payload returns a copy of the report ready to be marshalled, without the fields DefectDojo does not need.
// The Trivy Operator parser builds the finding service from the resource labels and DefectDojo closes
// findings only within one service, so the labels name the stable workload: a rollout then keeps one service.
func Payload(report *unstructured.Unstructured, workload Workload) map[string]any {
	obj := report.DeepCopy()
	unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")

	if kind := stableKinds[workload.Kind]; kind != "" {
		labels := obj.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}

		labels[LabelResourceKind] = kind
		labels[LabelResourceName] = workload.Name
		obj.SetLabels(labels)
	}

	return obj.Object
}

// stableKinds are the workload kinds that replace the resource kind of their reports.
var stableKinds = map[string]string{"deployment": kindDeployment, "cronjob": kindCronJob}

// Service returns the finding service DefectDojo derives from the report sent by Payload.
func Service(report *unstructured.Unstructured, workload Workload) string {
	labels := report.GetLabels()
	namespace := labels[LabelResourceNamespace]

	if namespace == "" {
		namespace = report.GetNamespace()
	}

	kind, name := labels[LabelResourceKind], labels[LabelResourceName]
	if stable := stableKinds[workload.Kind]; stable != "" {
		kind, name = stable, workload.Name
	}

	return namespace + "/" + kind + "/" + name + "/" + labels[LabelContainerName]
}
