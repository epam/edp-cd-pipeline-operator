// Package runtimesecurity re-imports the Trivy Operator reports of in-cluster Stage namespaces into DefectDojo.
package runtimesecurity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/defectdojo"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/security"
)

const (
	controllerName = "runtime-security"

	secretKeyURL   = "url"
	secretKeyToken = "token"

	reportsPageSize = 500
	// sweepInterval is the period of the check for runtime tests of Stages that no longer exist.
	sweepInterval = 6 * time.Hour
)

// DefectDojo is the part of the DefectDojo client the controller uses.
type DefectDojo interface {
	Reimport(ctx context.Context, r defectdojo.ReimportRequest) (defectdojo.ReimportResult, error)
	GetTest(ctx context.Context, testID int) (defectdojo.Test, error)
	ListTests(ctx context.Context, tag string) ([]defectdojo.Test, error)
	CloseTest(ctx context.Context, testID int) error
	CloseFindingsOutsideService(ctx context.Context, testID int, service string) (int, error)
}

// Connector returns a DefectDojo client for the given credentials.
type Connector func(url, token string) DefectDojo

// NewConnector binds the shared DefectDojo client state to the credentials of each call.
func NewConnector(c *defectdojo.Client) Connector {
	return func(url, token string) DefectDojo { return c.WithCredentials(url, token) }
}

// importKey identifies a DefectDojo test inside a Stage.
type importKey struct {
	product    string
	engagement string
	title      string
}

// importedReport is the report version a DefectDojo test currently reflects.
type importedReport struct {
	uid             types.UID
	resourceVersion string
	testID          int
}

// Reconciler imports Trivy Operator reports of Stage namespaces into DefectDojo.
//
// It also binds the reader ClusterRole to the operator ServiceAccount inside each Stage namespace it reads,
// after the ownership check. A failed bind is logged and retried after the poll interval, so the feature never
// blocks the provisioning of a Stage. The delete chain of the Stage controller removes the RoleBinding.
//
// DefectDojo, the Secret with its credentials and the Trivy Operator CRDs are optional:
// when one of them is missing or failing the Stage is retried after the poll interval and no error is returned.
// The reconciler adds no finalizer, so DefectDojo never blocks Stage deletion.
type Reconciler struct {
	client    client.Client
	namespace string
	config    Config
	connect   Connector
	now       func() time.Time

	mu sync.Mutex
	// imported holds, per Stage namespace, the report each test was last imported from.
	// It is lost on restart; the next pass re-imports every report and converges.
	imported map[string]map[importKey]importedReport
	// closed holds the tests that were closed because their workload is gone.
	closed map[int]struct{}
	// healed holds the tests whose findings of stale services were closed.
	healed map[int]struct{}
}

// NewReconciler returns a Reconciler for the Stages of the tenant in namespace.
func NewReconciler(k8sClient client.Client, namespace string, cfg Config, connect Connector) *Reconciler {
	return &Reconciler{
		client:    k8sClient,
		namespace: namespace,
		config:    cfg,
		connect:   connect,
		now:       time.Now,
		imported:  map[string]map[importKey]importedReport{},
		closed:    map[int]struct{}{},
		healed:    map[int]struct{}{},
	}
}

// SetupWithManager registers the Stage controller and the sweeper of tests of deleted Stages.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&cdPipeApi.Stage{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		WithOptions(controller.Options{MaxConcurrentReconciles: r.config.Concurrency}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to build runtime security controller: %w", err)
	}

	if r.config.ServiceAccount == "" || r.config.ReaderClusterRole == "" {
		ctrl.Log.WithName(controllerName).Info(
			"Reader ServiceAccount or ClusterRole is not configured, Stage namespace reads are not granted "+
				"and imports fail until both are set",
			"serviceAccountEnv", ServiceAccountEnv, "readerClusterRoleEnv", ReaderClusterRoleEnv)
	}

	if err := mgr.Add(manager.RunnableFunc(r.runSweeper)); err != nil {
		return fmt.Errorf("failed to add runtime security sweeper: %w", err)
	}

	return nil
}

// Reconcile imports the reports of an in-cluster Stage and closes the tests of deleted Stages.
//
// The reads of a Stage namespace are granted per namespace by this controller, only after the ownership check.
// Before the bind and any read the Stage must carry DNS label names and, where ownership is enforced,
// name a namespace that belongs to this tenant.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	stage := &cdPipeApi.Stage{}
	if err := r.client.Get(ctx, request.NamespacedName, stage); err != nil {
		if k8sErrors.IsNotFound(err) {
			r.closeOrphanedTests(ctx)

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get Stage: %w", err)
	}

	if stage.DeletionTimestamp != nil || stage.Spec.Namespace == "" || !util.RuntimeSecurityApplies(stage) {
		return reconcile.Result{}, nil
	}

	if field := invalidName(stage); field != "" {
		log.Info("Stage field is not an RFC 1123 DNS label, runtime security reports are not imported",
			"field", field)

		return reconcile.Result{}, nil
	}

	if !r.namespaceOwned(ctx, stage) {
		return reconcile.Result{RequeueAfter: r.config.PollInterval}, nil
	}

	proceed, err := r.ensureRoleBinding(ctx, stage)
	if err != nil {
		if k8sErrors.IsForbidden(err) {
			log.Info("Operator cannot bind the reader role yet, will retry", "namespace", stage.Spec.Namespace)
		} else {
			log.Error(err, "Failed to bind the runtime security reader role, will retry",
				"namespace", stage.Spec.Namespace)
		}

		return reconcile.Result{RequeueAfter: r.config.PollInterval}, nil
	}

	if !proceed {
		return reconcile.Result{}, nil
	}

	dd, ok := r.defectDojo(ctx)
	if !ok {
		return reconcile.Result{RequeueAfter: r.config.PollInterval}, nil
	}

	if err = r.syncStage(ctx, dd, stage); err != nil {
		if k8sErrors.IsForbidden(err) {
			log.Info("Operator cannot read the Stage namespace yet, will retry", "namespace", stage.Spec.Namespace)
		} else {
			log.Error(err, "Failed to import runtime security reports, will retry", "namespace", stage.Spec.Namespace)
		}
	}

	return reconcile.Result{RequeueAfter: r.config.PollInterval}, nil
}

// ensureRoleBinding creates the reader RoleBinding in the Stage namespace when it is missing;
// an existing RoleBinding is never updated.
// proceed is false when the Stage was deleted while the RoleBinding was created.
func (r *Reconciler) ensureRoleBinding(ctx context.Context, stage *cdPipeApi.Stage) (proceed bool, err error) {
	log := ctrl.LoggerFrom(ctx).WithValues("namespace", stage.Spec.Namespace)

	serviceAccount, role := r.config.ServiceAccount, r.config.ReaderClusterRole
	if serviceAccount == "" || role == "" {
		return true, nil
	}

	key := client.ObjectKey{Name: util.RuntimeSecurityRbName, Namespace: stage.Spec.Namespace}

	err = r.client.Get(ctx, key, &rbacApi.RoleBinding{})
	if err == nil {
		return true, nil
	}

	if !k8sErrors.IsNotFound(err) {
		return false, fmt.Errorf("failed to get rolebinding %s: %w", key.Name, err)
	}

	rb := &rbacApi.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
		Subjects: []rbacApi.Subject{{
			Kind:      rbacApi.ServiceAccountKind,
			Name:      serviceAccount,
			Namespace: stage.Namespace,
		}},
		RoleRef: rbacApi.RoleRef{APIGroup: rbacApi.GroupName, Kind: rbac.ClusterRoleKind, Name: role},
	}

	if err = r.client.Create(ctx, rb); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return true, nil
		}

		return false, fmt.Errorf("failed to create rolebinding %s: %w", key.Name, err)
	}

	log.Info("Runtime security reader role has been bound")

	return r.keepIfStageLives(ctx, stage, rb)
}

// keepIfStageLives keeps the RoleBinding just created only for a Stage confirmed live.
// When the Stage was deleted meanwhile the delete chain may already have run and would not remove it again;
// when the re-read fails the RoleBinding is removed too and the Get error is returned.
// The next pass of a live Stage creates the RoleBinding again.
func (r *Reconciler) keepIfStageLives(
	ctx context.Context,
	stage *cdPipeApi.Stage,
	rb *rbacApi.RoleBinding,
) (bool, error) {
	current := &cdPipeApi.Stage{}

	err := r.client.Get(ctx, client.ObjectKeyFromObject(stage), current)
	if err == nil && current.DeletionTimestamp == nil {
		return true, nil
	}

	if delErr := r.client.Delete(ctx, rb); delErr != nil && !k8sErrors.IsNotFound(delErr) {
		return false, fmt.Errorf("failed to delete rolebinding %s of an unconfirmed Stage: %w", rb.Name, delErr)
	}

	if err != nil && !k8sErrors.IsNotFound(err) {
		return false, fmt.Errorf("failed to get Stage: %w", err)
	}

	ctrl.LoggerFrom(ctx).Info("Stage was deleted while binding, the runtime security reader role has been unbound",
		"namespace", rb.Namespace)

	return false, nil
}

// invalidName returns the Stage field that is not an RFC 1123 DNS label, or an empty string.
// The names end up in DefectDojo tags and titles, where a comma or a dot would forge or blur a tag.
func invalidName(stage *cdPipeApi.Stage) string {
	fields := []struct{ name, value string }{
		{"spec.cdPipeline", stage.Spec.CdPipeline},
		{"spec.name", stage.Spec.Name},
		{"spec.namespace", stage.Spec.Namespace},
	}

	for _, f := range fields {
		if len(validation.IsDNS1123Label(f.value)) > 0 {
			return f.name
		}
	}

	return ""
}

// namespaceOwned reports whether the Stage namespace may be read. Where ownership is enforced
// the namespace must carry the tenant label; an unreadable or absent namespace does not pass.
func (r *Reconciler) namespaceOwned(ctx context.Context, stage *cdPipeApi.Stage) bool {
	if !util.OwnershipEnforced(stage) {
		return true
	}

	log := ctrl.LoggerFrom(ctx).WithValues("namespace", stage.Spec.Namespace)

	ns := &corev1.Namespace{}
	if err := r.client.Get(ctx, client.ObjectKey{Name: stage.Spec.Namespace}, ns); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("Stage namespace does not exist, runtime security reports are not imported")
		} else {
			log.Error(err, "Failed to read the Stage namespace, runtime security reports are not imported")
		}

		return false
	}

	if err := util.CheckNamespaceOwnership(ns, r.namespace); err != nil {
		log.Info("Stage namespace is not owned by the tenant, runtime security reports are not imported",
			"reason", err.Error())

		return false
	}

	return true
}

// defectDojo reads the integration Secret on every call so a rotated token takes effect without a restart.
func (r *Reconciler) defectDojo(ctx context.Context) (DefectDojo, bool) {
	log := ctrl.LoggerFrom(ctx)

	secret := &corev1.Secret{}
	if err := r.client.Get(ctx, client.ObjectKey{Namespace: r.namespace, Name: r.config.SecretName}, secret); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("DefectDojo integration Secret is absent, runtime security reports are not imported",
				"secret", r.config.SecretName)
		} else {
			log.Error(err, "Failed to read DefectDojo integration Secret", "secret", r.config.SecretName)
		}

		return nil, false
	}

	url, token := string(secret.Data[secretKeyURL]), string(secret.Data[secretKeyToken])
	if url == "" || token == "" {
		log.Info("DefectDojo integration Secret has no url or token, runtime security reports are not imported",
			"secret", r.config.SecretName)

		return nil, false
	}

	return r.connect(url, token), true
}

// syncStage imports the reports of the Stage namespace and closes the tests of workloads that are gone.
// Tests are closed only after every report was imported: a failed import hides which tests are still alive.
func (r *Reconciler) syncStage(ctx context.Context, dd DefectDojo, stage *cdPipeApi.Stage) error {
	log := ctrl.LoggerFrom(ctx)
	env := security.Environment{
		Tenant:     r.namespace,
		CDPipeline: stage.Spec.CdPipeline,
		Stage:      stage.Spec.Name,
		Namespace:  stage.Spec.Namespace,
	}

	reports, served, err := r.listReports(ctx, env.Namespace)
	if err != nil {
		return err
	}

	if !served {
		log.Info("Trivy Operator reports are not served by the cluster, nothing to import")

		return nil
	}

	codebases, err := r.codebasesByImage(ctx, stage.Namespace)
	if err != nil {
		return err
	}

	resolver := security.NewWorkloadResolver(r.client, env.Namespace)
	current := map[importKey]importedReport{}
	supersededTitles := map[string]struct{}{}

	var rejected error

	for i := range reports {
		report := &reports[i]
		if report.GetDeletionTimestamp() != nil {
			continue
		}

		labels := report.GetLabels()

		workload, state, resolveErr := resolver.Resolve(
			ctx, labels[security.LabelResourceKind], labels[security.LabelResourceName],
		)
		if resolveErr != nil {
			return resolveErr
		}

		if state == security.WorkloadGone {
			continue
		}

		target := security.ReportImport(env, report, workload, codebases)

		if state == security.WorkloadSuperseded {
			supersededTitles[target.TestTitle] = struct{}{}

			continue
		}

		key := importKey{product: target.Product, engagement: target.Engagement, title: target.TestTitle}

		if prev, ok := r.lastImport(env.Namespace, key); ok &&
			prev.uid == report.GetUID() && prev.resourceVersion == report.GetResourceVersion() {
			current[key] = prev

			continue
		}

		entry, importErr := r.importReport(ctx, dd, target, report, workload)
		if importErr != nil {
			if !defectdojo.IsRejectedReport(importErr) {
				return importErr
			}

			log.Error(importErr, "DefectDojo rejected a report", "report", report.GetName(), "kind", report.GetKind())

			rejected = errors.Join(rejected, importErr)

			continue
		}

		current[key] = entry

		r.recordImport(env.Namespace, key, entry)

		log.Info("Report has been imported", "report", report.GetName(), "kind", report.GetKind(),
			"product", target.Product, "engagement", target.Engagement, "test", entry.testID)
	}

	if rejected != nil {
		return rejected
	}

	r.keepImports(env.Namespace, current)

	return r.closeStaleTests(ctx, dd, env, current, supersededTitles)
}

func (r *Reconciler) importReport(
	ctx context.Context,
	dd DefectDojo,
	target security.Import,
	report *unstructured.Unstructured,
	workload security.Workload,
) (importedReport, error) {
	payload, err := json.Marshal(security.Payload(report, workload))
	if err != nil {
		return importedReport{}, fmt.Errorf("failed to marshal report %s: %w", report.GetName(), err)
	}

	res, err := dd.Reimport(ctx, target.Request(r.config.ProductType, r.now(), payload))
	if err != nil {
		return importedReport{}, fmt.Errorf("failed to import report %s: %w", report.GetName(), err)
	}

	if res.TestID == 0 {
		return importedReport{}, fmt.Errorf("defectdojo did not return a test for report %s", report.GetName())
	}

	r.closeStaleServices(ctx, dd, res.TestID, security.Service(report, workload))

	return importedReport{uid: report.GetUID(), resourceVersion: report.GetResourceVersion(), testID: res.TestID}, nil
}

// closeStaleServices closes, once per test and process, the findings written under another service:
// reports imported before their labels named the stable workload left one set of findings per ReplicaSet.
// Only a test of this tenant is touched.
// A failure is retried after the next import.
func (r *Reconciler) closeStaleServices(ctx context.Context, dd DefectDojo, testID int, service string) {
	if r.isHealed(testID) {
		return
	}

	test, err := dd.GetTest(ctx, testID)
	if err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "Failed to read the test of stale services", "test", testID)

		return
	}

	if !r.ownsTest(ctx, test) {
		return
	}

	closed, err := dd.CloseFindingsOutsideService(ctx, testID, service)
	if err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "Failed to close findings of a stale service", "test", testID)

		return
	}

	r.markHealed(testID)

	if closed > 0 {
		ctrl.LoggerFrom(ctx).Info("Findings of a stale service have been closed", "test", testID, "findings", closed)
	}
}

// listReports returns the reports of every served kind in the namespace.
// served is false when the cluster serves none of the Trivy Operator report kinds.
func (r *Reconciler) listReports(ctx context.Context, namespace string) ([]unstructured.Unstructured, bool, error) {
	var (
		reports []unstructured.Unstructured
		served  bool
	)

	for _, kind := range security.ReportKinds {
		gvk := security.ReportGVK(kind)
		base := []client.ListOption{client.InNamespace(namespace), client.Limit(reportsPageSize)}
		opts := base

		for {
			list := &unstructured.UnstructuredList{}
			list.SetGroupVersionKind(gvk.GroupVersion().WithKind(kind + "List"))

			if err := r.client.List(ctx, list, opts...); err != nil {
				if meta.IsNoMatchError(err) || k8sErrors.IsNotFound(err) {
					break
				}

				return nil, false, fmt.Errorf("failed to list %s: %w", kind, err)
			}

			served = true

			for i := range list.Items {
				// The item kind is not always set on a list read.
				list.Items[i].SetGroupVersionKind(gvk)
				reports = append(reports, list.Items[i])
			}

			if list.GetContinue() == "" {
				break
			}

			opts = append(slices.Clone(base), client.Continue(list.GetContinue()))
		}
	}

	return reports, served, nil
}

// codebasesByImage maps the normalised image of each CodebaseImageStream to its codebase.
func (r *Reconciler) codebasesByImage(ctx context.Context, namespace string) (map[string]string, error) {
	streams := &codebaseApi.CodebaseImageStreamList{}
	if err := r.client.List(ctx, streams, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list CodebaseImageStreams: %w", err)
	}

	codebases := make(map[string]string, len(streams.Items))

	for _, stream := range streams.Items {
		if stream.Spec.ImageName == "" || stream.Spec.Codebase == "" {
			continue
		}

		image := security.NormalizeImage(stream.Spec.ImageName)
		if _, ok := codebases[image]; !ok {
			codebases[image] = stream.Spec.Codebase
		}
	}

	return codebases, nil
}

// closeStaleTests closes the runtime tests of the Stage and its namespace
// that no longer reflect a running workload container.
// A test whose Deployment has only superseded reports so far (a rollout is in progress) stays open.
func (r *Reconciler) closeStaleTests(
	ctx context.Context,
	dd DefectDojo,
	env security.Environment,
	current map[importKey]importedReport,
	supersededTitles map[string]struct{},
) error {
	live := make(map[int]struct{}, len(current))
	for _, entry := range current {
		live[entry.testID] = struct{}{}
	}

	envTag := security.EnvTag(env.CDPipeline, env.Stage)
	namespaceTag := security.NamespaceTag(env.Namespace)

	tests, err := dd.ListTests(ctx, envTag)
	if err != nil {
		return fmt.Errorf("failed to list DefectDojo tests of the Stage: %w", err)
	}

	for _, test := range tests {
		if !isRuntimeTestOf(test, r.namespace) || !test.HasTag(envTag) || !test.HasTag(namespaceTag) {
			continue
		}

		if _, ok := live[test.ID]; ok {
			r.markOpen(test.ID)

			continue
		}

		if _, ok := supersededTitles[test.Title]; ok {
			continue
		}

		if err = r.closeTest(ctx, dd, test); err != nil {
			return err
		}
	}

	return nil
}

// closeOrphanedTests closes the runtime tests of this tenant whose Stage no longer exists or left the cluster.
func (r *Reconciler) closeOrphanedTests(ctx context.Context) {
	log := ctrl.LoggerFrom(ctx)

	dd, ok := r.defectDojo(ctx)
	if !ok {
		return
	}

	stages := &cdPipeApi.StageList{}
	if err := r.client.List(ctx, stages, client.InNamespace(r.namespace)); err != nil {
		log.Error(err, "Failed to list Stages, runtime tests of deleted Stages are not closed")

		return
	}

	// A test is kept while a Stage of the same environment and namespace exists.
	// A test without a namespace tag is kept while a Stage of the environment exists.
	existing := make(map[string]map[string]struct{}, len(stages.Items))

	for i := range stages.Items {
		stage := &stages.Items[i]
		if stage.InCluster() {
			envTag := security.EnvTag(stage.Spec.CdPipeline, stage.Spec.Name)

			if existing[envTag] == nil {
				existing[envTag] = map[string]struct{}{}
			}

			existing[envTag][security.NamespaceTag(stage.Spec.Namespace)] = struct{}{}
		}
	}

	tests, err := dd.ListTests(ctx, security.TenantTag(r.namespace))
	if err != nil {
		log.Error(err, "Failed to list DefectDojo tests, runtime tests of deleted Stages are not closed")

		return
	}

	for _, test := range tests {
		if !isRuntimeTestOf(test, r.namespace) {
			continue
		}

		env, ok := test.TagValue(security.TagPrefixEnv)
		if !ok {
			continue
		}

		if namespaces, found := existing[security.TagPrefixEnv+":"+env]; found {
			namespace, hasNamespace := test.TagValue(security.TagPrefixNamespace)
			if _, same := namespaces[security.NamespaceTag(namespace)]; !hasNamespace || same {
				continue
			}
		}

		if err = r.closeTest(ctx, dd, test); err != nil {
			log.Error(err, "Failed to close runtime test of a deleted Stage, will retry")

			return
		}
	}
}

func (r *Reconciler) closeTest(ctx context.Context, dd DefectDojo, test defectdojo.Test) error {
	if r.isClosed(test.ID) || !r.ownsTest(ctx, test) {
		return nil
	}

	if err := dd.CloseTest(ctx, test.ID); err != nil {
		return fmt.Errorf("failed to close DefectDojo test %d: %w", test.ID, err)
	}

	r.markClosed(test.ID)

	ctrl.LoggerFrom(ctx).Info("Runtime test has been closed", "test", test.ID, "title", test.Title)

	return nil
}

// runSweeper closes the runtime tests of deleted Stages at start and then periodically,
// which also covers a Stage deleted while the operator was down.
func (r *Reconciler) runSweeper(ctx context.Context) error {
	ctx = ctrl.LoggerInto(ctx, ctrl.Log.WithName(controllerName).WithName("sweeper"))

	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		r.closeOrphanedTests(ctx)

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// ownsTest reports whether the test carries this operator's tenant tag; findings of any other test stay untouched.
func (r *Reconciler) ownsTest(ctx context.Context, test defectdojo.Test) bool {
	if test.HasTag(security.TenantTag(r.namespace)) {
		return true
	}

	ctrl.LoggerFrom(ctx).Info("Test belongs to another tenant, skipping", "test", test.ID, "title", test.Title)

	return false
}

func isRuntimeTestOf(test defectdojo.Test, tenant string) bool {
	return test.HasTag(security.TagScopeRuntime) && test.HasTag(security.TenantTag(tenant))
}

func (r *Reconciler) lastImport(namespace string, key importKey) (importedReport, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.imported[namespace][key]

	return entry, ok
}

func (r *Reconciler) recordImport(namespace string, key importKey, entry importedReport) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.imported[namespace] == nil {
		r.imported[namespace] = map[importKey]importedReport{}
	}

	r.imported[namespace][key] = entry
	delete(r.closed, entry.testID)
}

// keepImports forgets the tests that no report reflects any more.
func (r *Reconciler) keepImports(namespace string, current map[importKey]importedReport) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.imported[namespace] = current
}

func (r *Reconciler) isClosed(testID int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.closed[testID]

	return ok
}

func (r *Reconciler) markClosed(testID int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed[testID] = struct{}{}
}

func (r *Reconciler) markOpen(testID int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.closed, testID)
}

func (r *Reconciler) isHealed(testID int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.healed[testID]

	return ok
}

func (r *Reconciler) markHealed(testID int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.healed[testID] = struct{}{}
}
