package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func loadFixture(t *testing.T, name string) *unstructured.Unstructured {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	obj := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &obj))

	return &unstructured.Unstructured{Object: obj}
}

func testEnvironment() Environment {
	return Environment{Tenant: "krci", CDPipeline: "demo", Stage: "dev", Namespace: testNamespace}
}

func TestReportImport(t *testing.T) {
	codebases := map[string]string{
		"gitlab.127.0.0.1.nip.io:5050/krci/test-go-app": "test-go-app",
		"valkey/valkey": "cache",
	}

	tests := []struct {
		name     string
		fixture  string
		workload Workload
		want     Import
	}{
		{
			name:     "image of a codebase goes to the codebase product",
			fixture:  "vulnerabilityreport_replicaset.json",
			workload: Workload{"deployment", "test-go-app"},
			want: Import{
				Product:    "test-go-app",
				Engagement: "runtime:demo-dev",
				TestTitle:  "VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app",
				Version:    "main-20261110-194848",
				Codebase:   "test-go-app",
				Tags: []string{
					"krci-scope:runtime", "krci-tenant:krci", "krci-env:demo.dev", "krci-namespace:krci-demo-dev",
					"krci-workload:deployment.test-go-app", "krci-container:test-go-app", "krci-codebase:test-go-app",
				},
			},
		},
		{
			name:     "foreign image goes to the environment product",
			fixture:  "vulnerabilityreport_dockerhub.json",
			workload: Workload{"deployment", "defectdojo-primary"},
			want: Import{
				Product:    "env:demo-dev",
				Engagement: "runtime:demo-dev",
				TestTitle:  "VulnerabilityReport krci-demo-dev/deployment/defectdojo-primary/postgres",
				Version:    "16-alpine",
				Tags: []string{
					"krci-scope:runtime", "krci-tenant:krci", "krci-env:demo.dev", "krci-namespace:krci-demo-dev",
					"krci-workload:deployment.defectdojo-primary", "krci-container:postgres",
				},
			},
		},
		{
			name:     "whole image reference in the tag is matched and versioned by its tag",
			fixture:  "vulnerabilityreport_fullref_tag.json",
			workload: Workload{"statefulset", "defectdojo-valkey"},
			want: Import{
				Product:    "cache",
				Engagement: "runtime:demo-dev",
				TestTitle:  "VulnerabilityReport krci-demo-dev/statefulset/defectdojo-valkey/valkey",
				Version:    "9.0.0-alpine3.22",
				Codebase:   "cache",
				Tags: []string{
					"krci-scope:runtime", "krci-tenant:krci", "krci-env:demo.dev", "krci-namespace:krci-demo-dev",
					"krci-workload:statefulset.defectdojo-valkey", "krci-container:valkey", "krci-codebase:cache",
				},
			},
		},
		{
			name:     "exposed secret report has its own test in the same product",
			fixture:  "exposedsecretreport_replicaset.json",
			workload: Workload{"deployment", "test-go-app"},
			want: Import{
				Product:    "test-go-app",
				Engagement: "runtime:demo-dev",
				TestTitle:  "ExposedSecretReport krci-demo-dev/deployment/test-go-app/test-go-app",
				Version:    "main-20261110-194848",
				Codebase:   "test-go-app",
				Tags: []string{
					"krci-scope:runtime", "krci-tenant:krci", "krci-env:demo.dev", "krci-namespace:krci-demo-dev",
					"krci-workload:deployment.test-go-app", "krci-container:test-go-app", "krci-codebase:test-go-app",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReportImport(testEnvironment(), loadFixture(t, tt.fixture), tt.workload, codebases)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReportImport_VersionFallsBackToDigest(t *testing.T) {
	report := loadFixture(t, "vulnerabilityreport_replicaset.json")
	require.NoError(t, unstructured.SetNestedField(report.Object, "", "report", "artifact", "tag"))

	got := ReportImport(testEnvironment(), report, Workload{"deployment", "app"}, nil)

	assert.Equal(t, "sha256:9155b03ee4d9", got.Version)
}

func TestImport_Request(t *testing.T) {
	scanDate := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	i := Import{Product: "p", Engagement: "e", TestTitle: "t", Version: "v", Tags: []string{"a"}}

	req := i.Request("KubeRocketCI", scanDate, []byte("{}"))

	assert.Equal(t, "KubeRocketCI", req.ProductType)
	assert.Equal(t, "p", req.Product)
	assert.Equal(t, "e", req.Engagement)
	assert.Equal(t, "t", req.TestTitle)
	assert.Equal(t, "v", req.Version)
	assert.Equal(t, []string{"a"}, req.Tags)
	assert.Equal(t, scanDate, req.ScanDate)
	assert.Equal(t, []byte("{}"), req.Report)
}

func TestPayload(t *testing.T) {
	report := loadFixture(t, "vulnerabilityreport_replicaset.json")
	require.NoError(t, unstructured.SetNestedSlice(report.Object, []any{map[string]any{"manager": "x"}}, "metadata", "managedFields"))

	tests := []struct {
		name     string
		workload Workload
		kind     string
		resource string
		service  string
	}{
		{
			name:     "replicaset becomes its deployment",
			workload: Workload{"deployment", "test-go-app"},
			kind:     "Deployment", resource: "test-go-app",
			service: "krci-demo-dev/Deployment/test-go-app/test-go-app",
		},
		{
			name:     "job becomes its cronjob",
			workload: Workload{"cronjob", "nightly"},
			kind:     "CronJob", resource: "nightly",
			service: "krci-demo-dev/CronJob/nightly/test-go-app",
		},
		{
			name:     "other kinds are unchanged",
			workload: Workload{"statefulset", "db"},
			kind:     "ReplicaSet", resource: "test-go-app-9dc66c9c6",
			service: "krci-demo-dev/ReplicaSet/test-go-app-9dc66c9c6/test-go-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := Payload(report, tt.workload)
			sent := &unstructured.Unstructured{Object: payload}

			_, found, _ := unstructured.NestedFieldNoCopy(payload, "metadata", "managedFields")
			assert.False(t, found)
			assert.Equal(t, "VulnerabilityReport", payload["kind"])
			assert.Contains(t, payload, "report")
			assert.Equal(t, tt.kind, sent.GetLabels()[LabelResourceKind])
			assert.Equal(t, tt.resource, sent.GetLabels()[LabelResourceName])
			assert.Equal(t, "test-go-app", sent.GetLabels()[LabelContainerName])
			assert.Equal(t, tt.service, Service(report, tt.workload))
		})
	}

	_, found, _ := unstructured.NestedFieldNoCopy(report.Object, "metadata", "managedFields")
	assert.True(t, found, "the report is not modified")
	assert.Equal(t, "ReplicaSet", report.GetLabels()[LabelResourceKind])
}

func TestTags(t *testing.T) {
	assert.Equal(t, "krci-tenant:krci", TenantTag("krci"))
	assert.Equal(t, "krci-env:demo.dev", EnvTag("demo", "dev"))
}

func TestReportGVK(t *testing.T) {
	gvk := ReportGVK(KindExposedSecretReport)

	assert.Equal(t, "aquasecurity.github.io", gvk.Group)
	assert.Equal(t, "v1alpha1", gvk.Version)
	assert.Equal(t, "ExposedSecretReport", gvk.Kind)
}

func TestImportTitle_IsUniquePerNamespace(t *testing.T) {
	w := Workload{"deployment", "app"}

	assert.Equal(t, "VulnerabilityReport ns-a/deployment/app/web", ImportTitle("VulnerabilityReport", "ns-a", w, "web"))
	assert.NotEqual(t, ImportTitle("VulnerabilityReport", "ns-a", w, "web"), ImportTitle("VulnerabilityReport", "ns-b", w, "web"))
}

func TestNamespaceTag(t *testing.T) {
	assert.Equal(t, "krci-namespace:ns-a", NamespaceTag("ns-a"))
}
