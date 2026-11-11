package runtimesecurity

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/defectdojo"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/security"
)

const (
	tenant       = "krci"
	stageNS      = "krci-demo-dev"
	secretName   = "ci-defectdojo"
	pollInterval = 7 * time.Minute
	operatorSA   = "edp-cd-pipeline-operator"
	readerRole   = "edp-cd-pipeline-operator-krci-runtime-security"

	revisionAnnotation = "deployment.kubernetes.io/revision"
)

var scanDate = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

type fakeDefectDojo struct {
	mu          sync.Mutex
	reimports   []defectdojo.ReimportRequest
	reports     [][]byte
	closed      []int
	tests       []defectdojo.Test
	ids         map[string]int
	reimportErr func(defectdojo.ReimportRequest) error
	listErr     error
	listCalls   int
	healCalls   []healCall
	getCalls    []int
	healErr     error
}

type healCall struct {
	testID  int
	service string
}

func (f *fakeDefectDojo) Reimport(_ context.Context, r defectdojo.ReimportRequest) (defectdojo.ReimportResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.reimports = append(f.reimports, r)
	f.reports = append(f.reports, r.Report)

	if f.reimportErr != nil {
		if err := f.reimportErr(r); err != nil {
			return defectdojo.ReimportResult{}, err
		}
	}

	if f.ids == nil {
		f.ids = map[string]int{}
	}

	key := r.Product + "/" + r.Engagement + "/" + r.TestTitle
	if _, ok := f.ids[key]; !ok {
		f.ids[key] = 100 + len(f.ids)
	}

	return defectdojo.ReimportResult{TestID: f.ids[key]}, nil
}

func (f *fakeDefectDojo) GetTest(_ context.Context, testID int) (defectdojo.Test, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.getCalls = append(f.getCalls, testID)

	for _, t := range f.tests {
		if t.ID == testID {
			return t, nil
		}
	}

	return defectdojo.Test{ID: testID, Tags: []string{security.TagScopeRuntime, security.TenantTag(tenant)}}, nil
}

func (f *fakeDefectDojo) ListTests(_ context.Context, tag string) ([]defectdojo.Test, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.listCalls++

	if f.listErr != nil {
		return nil, f.listErr
	}

	var out []defectdojo.Test

	for _, t := range f.tests {
		if t.HasTag(tag) {
			out = append(out, t)
		}
	}

	return out, nil
}

func (f *fakeDefectDojo) CloseTest(_ context.Context, testID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed = append(f.closed, testID)

	return nil
}

func (f *fakeDefectDojo) CloseFindingsOutsideService(_ context.Context, testID int, service string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.healCalls = append(f.healCalls, healCall{testID: testID, service: service})

	return 3, f.healErr
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, cdPipeApi.AddToScheme(s))
	require.NoError(t, codebaseApi.AddToScheme(s))

	return s
}

func newStage(name, stage, namespace, cluster string) *cdPipeApi.Stage {
	return &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: tenant},
		Spec: cdPipeApi.StageSpec{
			CdPipeline:  "demo",
			Name:        stage,
			Namespace:   namespace,
			ClusterName: cluster,
		},
	}
}

func demoStage() *cdPipeApi.Stage {
	return newStage("demo-dev", "dev", stageNS, cdPipeApi.InCluster)
}

func integrationSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: tenant},
		Data:       map[string][]byte{"url": []byte("https://dd.example.com"), "token": []byte("t")},
	}
}

func codebaseImageStream(imageName, codebase string) *codebaseApi.CodebaseImageStream {
	return &codebaseApi.CodebaseImageStream{
		ObjectMeta: metav1.ObjectMeta{Name: codebase + "-main", Namespace: tenant},
		Spec:       codebaseApi.CodebaseImageStreamSpec{ImageName: imageName, Codebase: codebase},
	}
}

func fixture(t *testing.T, name string) *unstructured.Unstructured {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "pkg", "security", "testdata", name))
	require.NoError(t, err)

	obj := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &obj))

	u := &unstructured.Unstructured{Object: obj}
	u.SetNamespace(stageNS)
	u.SetUID(types.UID("uid-" + name))
	u.SetResourceVersion("100")

	return u
}

func goApp(t *testing.T) *unstructured.Unstructured {
	t.Helper()

	return fixture(t, "vulnerabilityreport_replicaset.json")
}

func goAppSecrets(t *testing.T) *unstructured.Unstructured {
	t.Helper()

	return fixture(t, "exposedsecretreport_replicaset.json")
}

func replicaSet(name, deployment, revision string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   stageNS,
			Annotations: map[string]string{revisionAnnotation: revision},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1", Kind: "Deployment", Name: deployment, Controller: ptr.To(true),
			}},
		},
		Spec: appsv1.ReplicaSetSpec{Replicas: ptr.To(int32(1))},
	}
}

func deployment(name, revision string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: name, Namespace: stageNS, Annotations: map[string]string{revisionAnnotation: revision},
	}}
}

// goAppObjects is a running go app: its Deployment on revision 1, the current ReplicaSet and the codebase.
func goAppObjects() []client.Object {
	return []client.Object{
		demoStage(),
		integrationSecret(),
		codebaseImageStream("gitlab.127.0.0.1.nip.io:5050/krci/test-go-app", "test-go-app"),
		deployment("test-go-app", "1"),
		replicaSet("test-go-app-9dc66c9c6", "test-go-app", "1"),
	}
}

type harness struct {
	t          *testing.T
	client     client.Client
	dd         *fakeDefectDojo
	reconciler *Reconciler
}

func newHarness(t *testing.T, objects []client.Object, funcs ...interceptor.Funcs) *harness {
	t.Helper()

	t.Setenv(platform.SecurityEnabledEnv, "true")

	hasNamespace := false

	for _, o := range objects {
		if _, ok := o.(*corev1.Namespace); ok {
			hasNamespace = true
		}
	}

	if !hasNamespace {
		objects = append(slices.Clone(objects), ownedNamespace(stageNS))
	}

	b := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(objects...)
	if len(funcs) > 0 {
		b = b.WithInterceptorFuncs(funcs[0])
	}

	c := b.Build()
	dd := &fakeDefectDojo{}

	r := NewReconciler(c, tenant, Config{
		SecretName: secretName, ProductType: "KubeRocketCI", PollInterval: pollInterval, Concurrency: 1,
		ServiceAccount: operatorSA, ReaderClusterRole: readerRole,
	}, func(string, string) DefectDojo { return dd })
	r.now = func() time.Time { return scanDate }

	return &harness{t: t, client: c, dd: dd, reconciler: r}
}

func (h *harness) reconcile() reconcile.Result {
	h.t.Helper()

	res, err := h.reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: tenant, Name: "demo-dev"},
	})
	require.NoError(h.t, err)

	return res
}

func (h *harness) setReport(report *unstructured.Unstructured) {
	h.t.Helper()

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(report.GroupVersionKind())

	err := h.client.Get(context.Background(), client.ObjectKeyFromObject(report), existing)
	if err == nil {
		report.SetResourceVersion(existing.GetResourceVersion())
		require.NoError(h.t, h.client.Update(context.Background(), report))

		return
	}

	report.SetResourceVersion("")
	require.NoError(h.t, h.client.Create(context.Background(), report))
}

func (h *harness) deleteReport(report *unstructured.Unstructured) {
	h.t.Helper()

	report.SetResourceVersion("")
	require.NoError(h.t, h.client.Delete(context.Background(), report))
}

func withReports(objects []client.Object, reports ...*unstructured.Unstructured) []client.Object {
	for _, r := range reports {
		objects = append(objects, r)
	}

	return objects
}

func runtimeTest(id int, title string) defectdojo.Test {
	return runtimeTestIn(id, title, "dev", stageNS)
}

func runtimeTestIn(id int, title, stage, namespace string) defectdojo.Test {
	return defectdojo.Test{
		ID:    id,
		Title: title,
		Tags: []string{
			security.TagScopeRuntime, security.TenantTag(tenant), security.EnvTag("demo", stage),
			security.NamespaceTag(namespace),
		},
	}
}

func namespaceObject(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels}}
}

func ownedNamespace(name string) *corev1.Namespace {
	return namespaceObject(name, map[string]string{util.TenantLabelName: tenant})
}

func TestReconcile_ImportsCodebaseReports(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t), goAppSecrets(t)))

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	require.Len(t, h.dd.reimports, 2)

	byTitle := map[string]defectdojo.ReimportRequest{}
	for _, r := range h.dd.reimports {
		byTitle[r.TestTitle] = r
	}

	vuln := byTitle["VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app"]
	assert.Equal(t, "KubeRocketCI", vuln.ProductType)
	assert.Equal(t, "test-go-app", vuln.Product)
	assert.Equal(t, "runtime:demo-dev", vuln.Engagement)
	assert.Equal(t, "main-20261110-194848", vuln.Version)
	assert.Equal(t, scanDate, vuln.ScanDate)
	assert.ElementsMatch(t, []string{
		"krci-scope:runtime", "krci-tenant:krci", "krci-env:demo.dev", "krci-namespace:krci-demo-dev",
		"krci-workload:deployment.test-go-app", "krci-container:test-go-app", "krci-codebase:test-go-app",
	}, vuln.Tags)

	secrets := byTitle["ExposedSecretReport krci-demo-dev/deployment/test-go-app/test-go-app"]
	assert.Equal(t, "test-go-app", secrets.Product)

	payload := map[string]any{}
	require.NoError(t, json.Unmarshal(vuln.Report, &payload))
	assert.Equal(t, "VulnerabilityReport", payload["kind"])
	assert.NotContains(t, payload["metadata"], "managedFields")
}

func TestReconcile_ForeignImageGoesToEnvironmentProduct(t *testing.T) {
	objects := []client.Object{
		demoStage(),
		integrationSecret(),
		deployment("defectdojo-primary", "4"),
		replicaSet("defectdojo-primary-55747b486", "defectdojo-primary", "4"),
	}

	h := newHarness(t, withReports(objects, fixture(t, "vulnerabilityreport_dockerhub.json")))

	h.reconcile()

	require.Len(t, h.dd.reimports, 1)
	assert.Equal(t, "env:demo-dev", h.dd.reimports[0].Product)
	assert.Equal(t, "runtime:demo-dev", h.dd.reimports[0].Engagement)
	assert.Equal(t, "VulnerabilityReport krci-demo-dev/deployment/defectdojo-primary/postgres", h.dd.reimports[0].TestTitle)
	assert.NotContains(t, h.dd.reimports[0].Tags, "krci-codebase:postgres")
}

func TestReconcile_MatchesImageReferenceInTag(t *testing.T) {
	objects := []client.Object{
		demoStage(),
		integrationSecret(),
		codebaseImageStream("docker.io/valkey/valkey", "cache"),
	}

	report := fixture(t, "vulnerabilityreport_fullref_tag.json")
	report.SetNamespace(stageNS)

	h := newHarness(t, withReports(objects, report))

	h.reconcile()

	require.Len(t, h.dd.reimports, 1)
	assert.Equal(t, "cache", h.dd.reimports[0].Product)
	assert.Equal(t, "9.0.0-alpine3.22", h.dd.reimports[0].Version)
	assert.Equal(t, "VulnerabilityReport krci-demo-dev/statefulset/defectdojo-valkey/valkey", h.dd.reimports[0].TestTitle)
}

func TestReconcile_SkipsReportOfOldRevision(t *testing.T) {
	objects := goAppObjects()
	objects[3] = deployment("test-go-app", "2")

	h := newHarness(t, withReports(objects, goApp(t)))
	h.dd.tests = []defectdojo.Test{runtimeTest(7, "VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app")}

	h.reconcile()

	assert.Empty(t, h.dd.reimports)
	assert.Empty(t, h.dd.closed, "the test of a Deployment that is rolling out stays open")
}

func TestReconcile_SkipsUnchangedReport(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)))

	h.reconcile()
	require.Len(t, h.dd.reimports, 1)

	h.reconcile()
	assert.Len(t, h.dd.reimports, 1, "an unchanged report is not imported again")

	updated := goApp(t)
	updated.SetResourceVersion("")
	h.setReport(updated)

	h.reconcile()
	assert.Len(t, h.dd.reimports, 2, "a changed report is imported")
}

func TestReconcile_ImportsAgainWhenAnotherReportOverwroteTheTest(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)))

	h.reconcile()
	require.Len(t, h.dd.reimports, 1)

	// Rollout to revision 2: the report of the new ReplicaSet takes over the test.
	rollout := replicaSet("test-go-app-new", "test-go-app", "2")
	require.NoError(t, h.client.Create(context.Background(), rollout))

	dep := &appsv1.Deployment{}
	require.NoError(t, h.client.Get(context.Background(), client.ObjectKey{Namespace: stageNS, Name: "test-go-app"}, dep))
	dep.Annotations[revisionAnnotation] = "2"
	require.NoError(t, h.client.Update(context.Background(), dep))

	next := goApp(t)
	next.SetName("replicaset-test-go-app-new-test-go-app")
	next.SetUID("uid-new")
	next.SetLabels(map[string]string{
		security.LabelResourceKind:  "ReplicaSet",
		security.LabelResourceName:  "test-go-app-new",
		security.LabelContainerName: "test-go-app",
	})
	next.SetResourceVersion("")
	h.setReport(next)

	h.reconcile()
	require.Len(t, h.dd.reimports, 2)

	// Rollback: the old ReplicaSet is current again and its unchanged report must be imported again.
	dep.Annotations[revisionAnnotation] = "1"
	require.NoError(t, h.client.Update(context.Background(), dep))

	require.NoError(t, h.client.Delete(context.Background(), rollout))

	h.reconcile()
	assert.Len(t, h.dd.reimports, 3)
}

func TestReconcile_DefectDojoServerErrorRequeuesWithoutError(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t), goAppSecrets(t)))
	h.dd.reimportErr = func(defectdojo.ReimportRequest) error { return &defectdojo.APIError{StatusCode: 500} }
	h.dd.tests = []defectdojo.Test{runtimeTest(7, "VulnerabilityReport krci-demo-dev/deployment/gone/gone")}

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Len(t, h.dd.reimports, 1, "the pass stops at the first failure")
	assert.Empty(t, h.dd.closed, "nothing is closed while an import failed")

	h.dd.reimportErr = nil

	h.reconcile()
	assert.Len(t, h.dd.reimports, 3, "the failed report is retried")
}

func TestReconcile_RejectedReportDoesNotBlockTheOthers(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t), goAppSecrets(t)))
	h.dd.reimportErr = func(r defectdojo.ReimportRequest) error {
		if r.TestTitle == "VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app" {
			return &defectdojo.APIError{StatusCode: 400}
		}

		return nil
	}
	h.dd.tests = []defectdojo.Test{runtimeTest(7, "VulnerabilityReport krci-demo-dev/deployment/gone/gone")}

	h.reconcile()

	assert.Len(t, h.dd.reimports, 2)
	assert.Empty(t, h.dd.closed)
}

func TestReconcile_ListTestsErrorRequeuesWithoutError(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)))
	h.dd.listErr = &defectdojo.APIError{StatusCode: 502}

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
}

func TestReconcile_NoOps(t *testing.T) {
	noMatch := interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if u, ok := list.(*unstructured.UnstructuredList); ok && u.GroupVersionKind().Group == security.TrivyGroup {
				return &meta.NoKindMatchError{
					GroupKind:        schema.GroupKind{Group: security.TrivyGroup, Kind: "VulnerabilityReport"},
					SearchedVersions: []string{security.TrivyVersion},
				}
			}

			return c.List(ctx, list, opts...)
		},
	}

	tests := []struct {
		name        string
		objects     []client.Object
		funcs       []interceptor.Funcs
		wantRequeue time.Duration
	}{
		{
			name:        "Trivy Operator CRDs are absent",
			objects:     goAppObjects(),
			funcs:       []interceptor.Funcs{noMatch},
			wantRequeue: pollInterval,
		},
		{
			name: "Secret is absent",
			objects: withReports([]client.Object{
				demoStage(), deployment("test-go-app", "1"), replicaSet("test-go-app-9dc66c9c6", "test-go-app", "1"),
			}, goApp(t)),
			wantRequeue: pollInterval,
		},
		{
			name: "Secret has no token",
			objects: withReports([]client.Object{
				demoStage(),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: tenant},
					Data:       map[string][]byte{"url": []byte("https://dd.example.com")},
				},
			}, goApp(t)),
			wantRequeue: pollInterval,
		},
		{
			name: "Stage of a remote cluster",
			objects: withReports(
				[]client.Object{newStage("demo-dev", "dev", stageNS, "remote"), integrationSecret()},
				goApp(t),
			),
		},
		{
			name: "Stage without a namespace",
			objects: withReports(
				[]client.Object{newStage("demo-dev", "dev", "", cdPipeApi.InCluster), integrationSecret()},
				goApp(t),
			),
		},
		{
			name: "Stage is being deleted",
			objects: withReports([]client.Object{
				func() client.Object {
					s := demoStage()
					s.DeletionTimestamp = &metav1.Time{Time: scanDate}
					s.Finalizers = []string{"envLabelDeletion"}

					return s
				}(),
				integrationSecret(),
			}, goApp(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, tt.objects, tt.funcs...)
			h.dd.tests = []defectdojo.Test{runtimeTest(7, "VulnerabilityReport krci-demo-dev/deployment/gone/gone")}

			res := h.reconcile()

			assert.Equal(t, tt.wantRequeue, res.RequeueAfter)
			assert.Empty(t, h.dd.reimports)
			assert.Empty(t, h.dd.closed)
			assert.Zero(t, h.dd.listCalls)
		})
	}
}

func TestReconcile_ClosesTestOfWorkloadThatIsGone(t *testing.T) {
	gone := runtimeTest(7, "VulnerabilityReport krci-demo-dev/deployment/removed/removed")
	foreignTenant := runtimeTest(8, "VulnerabilityReport krci-demo-dev/deployment/x/x")
	foreignTenant.Tags = []string{security.TagScopeRuntime, security.TenantTag("other"), security.EnvTag("demo", "dev")}
	ciTest := defectdojo.Test{ID: 9, Title: "Trivy Scan", Tags: []string{security.EnvTag("demo", "dev")}}

	h := newHarness(t, withReports(goAppObjects(), goApp(t)))
	h.dd.tests = []defectdojo.Test{
		gone,
		foreignTenant,
		ciTest,
		runtimeTest(10, "VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app"),
	}

	h.reconcile()

	// Test 10 has the title of the running workload but is not the test the report was imported into.
	assert.ElementsMatch(t, []int{7, 10}, h.dd.closed)

	h.reconcile()
	assert.ElementsMatch(t, []int{7, 10}, h.dd.closed, "a closed test is not closed again")
}

func TestReconcile_ClosesAllTestsWhenStageNamespaceHasNoReports(t *testing.T) {
	h := newHarness(t, goAppObjects())
	h.dd.tests = []defectdojo.Test{runtimeTest(7, "VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app")}

	h.reconcile()

	assert.Equal(t, []int{7}, h.dd.closed)
	assert.Empty(t, h.dd.reimports, "a test is closed without a re-import")
}

func TestReconcile_ReopenedTestIsClosedAgainWhenWorkloadGoesAgain(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)))
	h.dd.tests = []defectdojo.Test{runtimeTest(100, "VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app")}

	h.reconcile()
	assert.Empty(t, h.dd.closed, "the imported test is live")

	h.deleteReport(goApp(t))

	h.reconcile()
	assert.Equal(t, []int{100}, h.dd.closed)

	h.setReport(goApp(t))

	h.reconcile()
	require.Len(t, h.dd.reimports, 2, "the report is imported again into the closed test")

	h.deleteReport(goApp(t))

	h.reconcile()
	assert.Equal(t, []int{100, 100}, h.dd.closed)
}

func TestReconcile_StageDeletedClosesItsTests(t *testing.T) {
	objects := []client.Object{
		newStage("demo-qa", "qa", "krci-demo-qa", cdPipeApi.InCluster),
		newStage("demo-prod", "prod", "krci-demo-prod", "remote"),
		integrationSecret(),
	}

	h := newHarness(t, objects)
	h.dd.tests = []defectdojo.Test{
		runtimeTest(1, "VulnerabilityReport krci-demo-dev/deployment/a/a"),
		runtimeTestIn(2, "VulnerabilityReport krci-demo-qa/deployment/a/a", "qa", "krci-demo-qa"),
		runtimeTestIn(3, "VulnerabilityReport krci-demo-prod/deployment/a/a", "prod", "krci-demo-prod"),
		{ID: 4, Title: "CI", Tags: []string{security.TenantTag(tenant)}},
		runtimeTest(5, "VulnerabilityReport krci-demo-dev/deployment/a/a"),
	}
	h.dd.tests[4].Tags[1] = security.TenantTag("other")

	res := h.reconcile()

	assert.Zero(t, res.RequeueAfter)
	assert.Equal(t, []int{1, 3}, h.dd.closed, "tests of the deleted Stage and of the Stage that left the cluster")
	assert.Empty(t, h.dd.reimports)
}

func TestReconcile_StageDeletedWithoutSecretIsNoOp(t *testing.T) {
	h := newHarness(t, nil)

	h.reconcile()

	assert.Zero(t, h.dd.listCalls)
}

func TestReconcile_PagesThroughReports(t *testing.T) {
	var calls []string

	funcs := interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			u, ok := list.(*unstructured.UnstructuredList)
			if !ok || u.GroupVersionKind().Kind != "VulnerabilityReportList" {
				return c.List(ctx, list, opts...)
			}

			lo := &client.ListOptions{}
			lo.ApplyOptions(opts)

			calls = append(calls, lo.Continue)

			if lo.Continue == "" {
				u.Items = []unstructured.Unstructured{*goApp(t)}
				u.SetContinue("page-2")

				return nil
			}

			dockerhub := fixture(t, "vulnerabilityreport_dockerhub.json")
			u.Items = []unstructured.Unstructured{*dockerhub}

			return nil
		},
	}

	objects := []client.Object{
		demoStage(),
		integrationSecret(),
		deployment("test-go-app", "1"),
		replicaSet("test-go-app-9dc66c9c6", "test-go-app", "1"),
		deployment("defectdojo-primary", "1"),
		replicaSet("defectdojo-primary-55747b486", "defectdojo-primary", "1"),
	}

	h := newHarness(t, objects, funcs)

	h.reconcile()

	assert.Equal(t, []string{"", "page-2"}, calls)
	assert.Len(t, h.dd.reimports, 2)
}

func TestReconcile_ReportIsNotModified(t *testing.T) {
	report := goApp(t)
	report.SetManagedFields([]metav1.ManagedFieldsEntry{{Manager: "trivy-operator"}})

	h := newHarness(t, withReports(goAppObjects(), report))

	h.reconcile()

	require.Len(t, h.dd.reports, 1)
	assert.NotContains(t, string(h.dd.reports[0]), "managedFields")

	stored := &unstructured.Unstructured{}
	stored.SetGroupVersionKind(security.ReportGVK(security.KindVulnerabilityReport))
	require.NoError(t, h.client.Get(context.Background(), client.ObjectKeyFromObject(report), stored))
	assert.NotEmpty(t, stored.GetManagedFields())
}

func TestReconcile_ListErrorOfReportsIsNotAnError(t *testing.T) {
	funcs := interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if _, ok := list.(*unstructured.UnstructuredList); ok {
				return errors.New("forbidden")
			}

			return c.List(ctx, list, opts...)
		},
	}

	h := newHarness(t, goAppObjects(), funcs)

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Empty(t, h.dd.reimports)
	assert.Zero(t, h.dd.listCalls)
}

func TestReconcile_ConnectsWithSecretCredentials(t *testing.T) {
	var gotURL, gotToken string

	h := newHarness(t, withReports(goAppObjects(), goApp(t)))
	h.reconciler.connect = func(url, token string) DefectDojo {
		gotURL, gotToken = url, token

		return h.dd
	}

	h.reconcile()

	assert.Equal(t, "https://dd.example.com", gotURL)
	assert.Equal(t, "t", gotToken)
}

func sentLabels(t *testing.T, raw []byte) map[string]any {
	t.Helper()

	payload := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &payload))

	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)

	labels, ok := metadata["labels"].(map[string]any)
	require.True(t, ok)

	return labels
}

func TestReconcile_SendsStableWorkloadLabels(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)))

	h.reconcile()

	require.Len(t, h.dd.reports, 1)

	labels := sentLabels(t, h.dd.reports[0])
	assert.Equal(t, "Deployment", labels[security.LabelResourceKind])
	assert.Equal(t, "test-go-app", labels[security.LabelResourceName])
	assert.Equal(t, "VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app", h.dd.reimports[0].TestTitle)
	assert.Contains(t, h.dd.reimports[0].Tags, "krci-workload:deployment.test-go-app")
}

func TestReconcile_ClosesFindingsOfStaleServicesOncePerTest(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)))

	h.reconcile()
	require.Len(t, h.dd.healCalls, 1)
	assert.Equal(t, healCall{testID: 100, service: "krci-demo-dev/Deployment/test-go-app/test-go-app"}, h.dd.healCalls[0])

	updated := goApp(t)
	h.setReport(updated)

	h.reconcile()
	require.Len(t, h.dd.reimports, 2)
	assert.Len(t, h.dd.healCalls, 1, "the steady state does not list findings")
}

func TestReconcile_RetriesStaleServiceCloseAfterFailure(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)))
	h.dd.healErr = &defectdojo.APIError{StatusCode: 500}

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	require.Len(t, h.dd.reimports, 1)
	require.Len(t, h.dd.healCalls, 1)

	h.dd.healErr = nil
	h.setReport(goApp(t))

	h.reconcile()
	assert.Len(t, h.dd.healCalls, 2)

	h.setReport(goApp(t))

	h.reconcile()
	assert.Len(t, h.dd.healCalls, 2)
}

func TestReconcile_CronJobRunsShareOneService(t *testing.T) {
	cronJob := func(name string, age time.Duration) *batchv1.Job {
		return &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         stageNS,
			CreationTimestamp: metav1.NewTime(scanDate.Add(-age)),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "batch/v1", Kind: "CronJob", Name: "nightly", Controller: ptr.To(true),
			}},
		}}
	}

	report := func(job string) *unstructured.Unstructured {
		r := goApp(t)
		r.SetName("job-" + job + "-app")
		r.SetUID(types.UID("uid-" + job))
		r.SetLabels(map[string]string{
			security.LabelResourceKind:      "Job",
			security.LabelResourceName:      job,
			security.LabelResourceNamespace: stageNS,
			security.LabelContainerName:     "app",
		})

		return r
	}

	objects := []client.Object{
		demoStage(), integrationSecret(), cronJob("nightly-1", time.Hour), cronJob("nightly-2", 0),
	}

	h := newHarness(t, withReports(objects, report("nightly-1"), report("nightly-2")))

	h.reconcile()

	require.Len(t, h.dd.reimports, 1, "only the newest run is imported")
	assert.Equal(t, "VulnerabilityReport krci-demo-dev/cronjob/nightly/app", h.dd.reimports[0].TestTitle)

	labels := sentLabels(t, h.dd.reports[0])
	assert.Equal(t, "CronJob", labels[security.LabelResourceKind])
	assert.Equal(t, "nightly", labels[security.LabelResourceName])
	assert.Equal(t, "krci-demo-dev/CronJob/nightly/app", h.dd.healCalls[0].service)
}

func TestReconcile_SkipsStageWithNonDNSLabelNames(t *testing.T) {
	tests := []struct {
		name  string
		stage *cdPipeApi.Stage
	}{
		{name: "comma in the stage name", stage: newStage("demo-dev", "dev,krci-env:other.prod", stageNS, cdPipeApi.InCluster)},
		{name: "dot in the stage name", stage: newStage("demo-dev", "dev.1", stageNS, cdPipeApi.InCluster)},
		{name: "upper case in the stage name", stage: newStage("demo-dev", "Dev", stageNS, cdPipeApi.InCluster)},
		{
			name: "invalid CD pipeline name",
			stage: func() *cdPipeApi.Stage {
				s := demoStage()
				s.Spec.CdPipeline = "de.mo"

				return s
			}(),
		},
		{name: "invalid namespace name", stage: newStage("demo-dev", "dev", "krci-demo-dev,x", cdPipeApi.InCluster)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, withReports([]client.Object{tt.stage, integrationSecret()}, goApp(t)))
			h.dd.tests = []defectdojo.Test{runtimeTest(7, "VulnerabilityReport krci-demo-dev/deployment/gone/gone")}

			res := h.reconcile()

			assert.Zero(t, res.RequeueAfter)
			assert.Empty(t, h.dd.reimports)
			assert.Empty(t, h.dd.closed)
			assert.Zero(t, h.dd.listCalls)
		})
	}
}

func TestReconcile_SkipsNamespaceThatIsNotOwned(t *testing.T) {
	tests := []struct {
		name    string
		objects []client.Object
	}{
		{name: "unlabeled namespace", objects: []client.Object{namespaceObject(stageNS, nil)}},
		{
			name:    "namespace of another tenant",
			objects: []client.Object{namespaceObject(stageNS, map[string]string{util.TenantLabelName: "other"})},
		},
		{name: "namespace is absent", objects: []client.Object{ownedNamespace("unrelated")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := append(withReports(goAppObjects(), goApp(t), goAppSecrets(t)), tt.objects...)
			h := newHarness(t, objects)
			h.dd.tests = []defectdojo.Test{runtimeTest(7, "VulnerabilityReport krci-demo-dev/deployment/gone/gone")}

			res := h.reconcile()

			assert.Equal(t, pollInterval, res.RequeueAfter)
			assert.Empty(t, h.dd.reimports, "nothing of the namespace is imported")
			assert.Empty(t, h.dd.closed)
			assert.Zero(t, h.dd.listCalls)
		})
	}
}

func TestReconcile_FailsClosedWhenNamespaceCannotBeRead(t *testing.T) {
	funcs := interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*corev1.Namespace); ok {
				return errors.New("apiserver unavailable")
			}

			return c.Get(ctx, key, obj, opts...)
		},
	}

	h := newHarness(t, withReports(goAppObjects(), goApp(t)), funcs)

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Empty(t, h.dd.reimports)
	assert.Zero(t, h.dd.listCalls)
}

func TestReconcile_ReadsUnlabeledNamespaceWhereOwnershipIsNotEnforced(t *testing.T) {
	t.Setenv(platform.TenancyEngineEnv, platform.TenancyEngineCapsule)

	h := newHarness(t, withReports(goAppObjects(), goApp(t)), interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*corev1.Namespace); ok {
				return errors.New("the namespace must not be read")
			}

			return c.Get(ctx, key, obj, opts...)
		},
	})

	h.reconcile()

	assert.Len(t, h.dd.reimports, 1)
}

func TestReconcile_ForbiddenReadIsSkippedUntilRoleBindingExists(t *testing.T) {
	forbidden := k8sErrors.NewForbidden(schema.GroupResource{Group: security.TrivyGroup, Resource: "vulnerabilityreports"}, "", errors.New("no"))
	denied := true

	funcs := interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if _, ok := list.(*unstructured.UnstructuredList); ok && denied {
				return forbidden
			}

			return c.List(ctx, list, opts...)
		},
	}

	h := newHarness(t, withReports(goAppObjects(), goApp(t)), funcs)
	h.dd.tests = []defectdojo.Test{runtimeTest(7, "VulnerabilityReport krci-demo-dev/deployment/gone/gone")}

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Empty(t, h.dd.reimports)
	assert.Empty(t, h.dd.closed, "a Forbidden read hides which workloads are alive")
	assert.Zero(t, h.dd.listCalls)

	denied = false

	h.reconcile()
	assert.Len(t, h.dd.reimports, 1)
}

func TestReconcile_ShadowStageClosesNothingOutsideItsNamespace(t *testing.T) {
	const foreignNS = "kube-system"

	realTest := runtimeTest(100, "VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app")
	foreignTest := runtimeTestIn(2, "VulnerabilityReport kube-system/deployment/x/x", "dev", foreignNS)

	// A second Stage of the same CD pipeline and name that targets another owned namespace.
	shadow := newStage("shadow", "dev", "krci-shadow", cdPipeApi.InCluster)

	objects := append(withReports(goAppObjects(), goApp(t)), shadow, ownedNamespace(stageNS), ownedNamespace("krci-shadow"))

	h := newHarness(t, objects)
	h.dd.tests = []defectdojo.Test{realTest, foreignTest}

	// The real Stage keeps its test and does not touch a test of another namespace.
	h.reconcile()
	assert.Empty(t, h.dd.closed)

	// The shadow Stage has no reports and no tests in its namespace: nothing of the real Stage closes.
	_, err := h.reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: tenant, Name: "shadow"},
	})
	require.NoError(t, err)
	assert.Empty(t, h.dd.closed, "tests of the real environment stay open")
}

func TestReconcile_OrphanSweepIsScopedByNamespace(t *testing.T) {
	// The Stage demo.dev exists, but only in krci-demo-dev: a test of demo.dev in another namespace is an orphan.
	objects := []client.Object{demoStage(), integrationSecret(), newStage("gone", "qa", "krci-demo-qa", cdPipeApi.InCluster)}

	h := newHarness(t, objects)
	h.dd.tests = []defectdojo.Test{
		runtimeTest(1, "VulnerabilityReport krci-demo-dev/deployment/a/a"),
		runtimeTestIn(2, "VulnerabilityReport old-ns/deployment/a/a", "dev", "old-ns"),
		{
			ID:    3,
			Title: "VulnerabilityReport deployment/a/a",
			Tags:  []string{security.TagScopeRuntime, security.TenantTag(tenant), security.EnvTag("demo", "dev")},
		},
	}

	h.reconciler.closeOrphanedTests(context.Background())

	assert.Equal(t, []int{2}, h.dd.closed, "a test without a namespace tag is kept while its environment exists")
}

func TestReconcile_DoesNotCloseTestsWithoutTenantTag(t *testing.T) {
	other := defectdojo.Test{
		ID:    5,
		Title: "VulnerabilityReport krci-demo-dev/deployment/gone/gone",
		Tags: []string{
			security.TagScopeRuntime, security.TenantTag("other"), security.EnvTag("demo", "dev"),
			security.NamespaceTag(stageNS),
		},
	}

	h := newHarness(t, goAppObjects())

	require.NoError(t, h.reconciler.closeTest(context.Background(), h.dd, other))
	assert.Empty(t, h.dd.closed)

	require.NoError(t, h.reconciler.closeTest(context.Background(), h.dd, runtimeTest(6, other.Title)))
	assert.Equal(t, []int{6}, h.dd.closed)
}

func TestReconcile_DoesNotCloseFindingsOfAnotherTenantsTest(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)))
	// The test the import lands in belongs to another tenant.
	h.dd.tests = []defectdojo.Test{{
		ID:    100,
		Title: "VulnerabilityReport krci-demo-dev/deployment/test-go-app/test-go-app",
		Tags:  []string{security.TagScopeRuntime, security.TenantTag("other")},
	}}

	h.reconcile()

	require.Len(t, h.dd.reimports, 1)
	assert.Equal(t, []int{100}, h.dd.getCalls)
	assert.Empty(t, h.dd.healCalls, "findings of another tenant's test are not closed")
	assert.Empty(t, h.dd.closed)
}

func (h *harness) roleBinding() (*rbacApi.RoleBinding, error) {
	h.t.Helper()

	rb := &rbacApi.RoleBinding{}
	err := h.client.Get(context.Background(), client.ObjectKey{Name: util.RuntimeSecurityRbName, Namespace: stageNS}, rb)

	return rb, err
}

func isRoleBinding(obj client.Object) bool {
	_, ok := obj.(*rbacApi.RoleBinding)

	return ok
}

// createThenDeleteStage deletes the Stage right after its RoleBinding was created.
func createThenDeleteStage(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
	if err := c.Create(ctx, obj, opts...); err != nil || !isRoleBinding(obj) {
		return err
	}

	return c.Delete(ctx, demoStage())
}

func TestReconcile_BindsReaderRole(t *testing.T) {
	tests := []struct {
		name        string
		env         func(t *testing.T)
		config      func(c *Config)
		objects     []client.Object
		wantBinding bool
		wantImports int
	}{
		{
			name:        "owned in-cluster Stage",
			objects:     withReports(goAppObjects(), goApp(t)),
			wantBinding: true,
			wantImports: 1,
		},
		{
			name:        "Capsule in-cluster Stage binds without the label check",
			env:         func(t *testing.T) { t.Setenv(platform.TenancyEngineEnv, platform.TenancyEngineCapsule) },
			objects:     append(withReports(goAppObjects(), goApp(t)), namespaceObject(stageNS, nil)),
			wantBinding: true,
			wantImports: 1,
		},
		{
			name:    "unlabeled namespace",
			objects: append(withReports(goAppObjects(), goApp(t)), namespaceObject(stageNS, nil)),
		},
		{
			name: "namespace of another tenant",
			objects: append(withReports(goAppObjects(), goApp(t)),
				namespaceObject(stageNS, map[string]string{util.TenantLabelName: "other"})),
		},
		{
			name:    "absent namespace",
			objects: append(withReports(goAppObjects(), goApp(t)), ownedNamespace("unrelated")),
		},
		{
			name: "invalid names",
			objects: withReports(
				[]client.Object{newStage("demo-dev", "Dev", stageNS, cdPipeApi.InCluster), integrationSecret()}, goApp(t),
			),
		},
		{
			name: "external cluster",
			objects: withReports(
				[]client.Object{newStage("demo-dev", "dev", stageNS, "remote"), integrationSecret()}, goApp(t),
			),
		},
		{
			name:    "OpenShift",
			env:     func(t *testing.T) { t.Setenv(platform.TypeEnv, platform.Openshift) },
			objects: withReports(goAppObjects(), goApp(t)),
		},
		{
			name:    "feature disabled",
			env:     func(t *testing.T) { t.Setenv(platform.SecurityEnabledEnv, "false") },
			objects: withReports(goAppObjects(), goApp(t)),
		},
		{
			name:        "empty ServiceAccount config skips the bind and keeps the import",
			config:      func(c *Config) { c.ServiceAccount = "" },
			objects:     withReports(goAppObjects(), goApp(t)),
			wantImports: 1,
		},
		{
			name:        "empty ClusterRole config skips the bind and keeps the import",
			config:      func(c *Config) { c.ReaderClusterRole = "" },
			objects:     withReports(goAppObjects(), goApp(t)),
			wantImports: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, tt.objects)

			if tt.env != nil {
				tt.env(t)
			}

			if tt.config != nil {
				tt.config(&h.reconciler.config)
			}

			h.reconcile()

			rb, err := h.roleBinding()
			if !tt.wantBinding {
				assert.True(t, k8sErrors.IsNotFound(err), "no RoleBinding is created")
			} else {
				require.NoError(t, err)
				assert.Equal(t, rbacApi.RoleRef{
					APIGroup: rbacApi.GroupName, Kind: rbac.ClusterRoleKind, Name: readerRole,
				}, rb.RoleRef)
				assert.Equal(t, []rbacApi.Subject{{
					Kind: rbacApi.ServiceAccountKind, Name: operatorSA, Namespace: tenant,
				}}, rb.Subjects)
			}

			assert.Len(t, h.dd.reimports, tt.wantImports)
		})
	}
}

func TestReconcile_KeepsExistingRoleBinding(t *testing.T) {
	existing := &rbacApi.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: util.RuntimeSecurityRbName, Namespace: stageNS},
		Subjects:   []rbacApi.Subject{{Kind: rbacApi.ServiceAccountKind, Name: "other", Namespace: tenant}},
		RoleRef:    rbacApi.RoleRef{APIGroup: rbacApi.GroupName, Kind: rbac.ClusterRoleKind, Name: "other"},
	}

	h := newHarness(t, append(withReports(goAppObjects(), goApp(t)), existing), interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if isRoleBinding(obj) {
				return errors.New("the RoleBinding must not be created")
			}

			return c.Create(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if isRoleBinding(obj) {
				return errors.New("the RoleBinding must not be updated")
			}

			return c.Update(ctx, obj, opts...)
		},
	})

	h.reconcile()

	rb, err := h.roleBinding()
	require.NoError(t, err)
	assert.Equal(t, existing.Subjects, rb.Subjects)
	assert.Equal(t, existing.RoleRef, rb.RoleRef)
	assert.Len(t, h.dd.reimports, 1)
}

func TestReconcile_RoleBindingErrorDoesNotFailTheStage(t *testing.T) {
	forbidden := k8sErrors.NewForbidden(schema.GroupResource{Group: rbacApi.GroupName, Resource: "rolebindings"}, "", errors.New("no"))
	denied := true

	h := newHarness(t, withReports(goAppObjects(), goApp(t)), interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if isRoleBinding(obj) && denied {
				return forbidden
			}

			return c.Create(ctx, obj, opts...)
		},
	})

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Empty(t, h.dd.reimports)
	assert.Zero(t, h.dd.listCalls)

	_, err := h.roleBinding()
	assert.True(t, k8sErrors.IsNotFound(err))

	denied = false

	res = h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Len(t, h.dd.reimports, 1)

	_, err = h.roleBinding()
	assert.NoError(t, err)
}

func TestReconcile_RemovesRoleBindingWhenStageIsDeletedDuringBind(t *testing.T) {
	tests := []struct {
		name  string
		stage func() *cdPipeApi.Stage
	}{
		{name: "Stage is gone", stage: demoStage},
		{
			name: "Stage is being deleted",
			stage: func() *cdPipeApi.Stage {
				s := demoStage()
				s.Finalizers = []string{"envLabelDeletion"}

				return s
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := append(withReports(goAppObjects()[1:], goApp(t)), tt.stage())

			h := newHarness(t, objects, interceptor.Funcs{Create: createThenDeleteStage})

			res := h.reconcile()

			assert.Zero(t, res.RequeueAfter)
			assert.Empty(t, h.dd.reimports)

			_, err := h.roleBinding()
			assert.True(t, k8sErrors.IsNotFound(err), "the RoleBinding of the deleted Stage is removed")
		})
	}
}

func TestReconcile_RoleBindingAlreadyExistsOnCreateIsNotAnError(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)), interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if isRoleBinding(obj) {
				return k8sErrors.NewAlreadyExists(schema.GroupResource{Group: rbacApi.GroupName, Resource: "rolebindings"}, obj.GetName())
			}

			return c.Create(ctx, obj, opts...)
		},
	})

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Len(t, h.dd.reimports, 1)
}

func TestReconcile_RoleBindingLookupErrorSkipsTheStagePass(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)), interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if isRoleBinding(obj) {
				return errors.New("apiserver unavailable")
			}

			return c.Get(ctx, key, obj, opts...)
		},
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if isRoleBinding(obj) {
				return errors.New("the RoleBinding must not be created")
			}

			return c.Create(ctx, obj, opts...)
		},
	})

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Empty(t, h.dd.reimports)
	assert.Zero(t, h.dd.listCalls)
}

func TestReconcile_RoleBindingIsRemovedWhenTheStageCannotBeConfirmed(t *testing.T) {
	stageReads := 0

	h := newHarness(t, withReports(goAppObjects(), goApp(t)), interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*cdPipeApi.Stage); ok {
				stageReads++

				if stageReads == 2 {
					return errors.New("apiserver unavailable")
				}
			}

			return c.Get(ctx, key, obj, opts...)
		},
	})

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Empty(t, h.dd.reimports)
	assert.Zero(t, h.dd.listCalls)

	_, err := h.roleBinding()
	assert.True(t, k8sErrors.IsNotFound(err), "a RoleBinding of an unconfirmed Stage is removed")
}

func TestReconcile_FailedCleanupOfRoleBindingDoesNotFailTheStage(t *testing.T) {
	h := newHarness(t, withReports(goAppObjects(), goApp(t)), interceptor.Funcs{
		Create: createThenDeleteStage,
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			if isRoleBinding(obj) {
				return errors.New("apiserver unavailable")
			}

			return c.Delete(ctx, obj, opts...)
		},
	})

	res := h.reconcile()

	assert.Equal(t, pollInterval, res.RequeueAfter)
	assert.Empty(t, h.dd.reimports)
	assert.Zero(t, h.dd.listCalls)
}
