package chart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

const (
	operatorUser = "operator"
	pipelineName = "app"
)

// verb is the request that carries the controller reference.
type verb string

const (
	verbCreate verb = "create"
	verbUpdate verb = "update"
	verbPatch  verb = "patch"
)

type roleDoc struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Rules []rbacv1.PolicyRule `json:"rules"`
}

// OwnerReferencesPermissionEnforcement, on by default on OpenShift, admits a blockOwnerDeletion
// ownerReference only from a user who may update the owner's finalizers, and an ownerReference
// change on an existing object only from a user who may delete that object.
func TestOperatorRole_AdmitsControllerReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("testdata", "codebaseimagestreams.v2.edp.epam.com.yaml"),
		},
		ErrorIfCRDPathMissing: true,
	}
	testEnv.ControlPlane.GetAPIServer().Configure().
		Append("enable-admission-plugins", "OwnerReferencesPermissionEnforcement")

	cfg, err := testEnv.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, testEnv.Stop())
	})

	admin, err := client.New(cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)

	user, err := testEnv.AddUser(envtest.User{Name: operatorUser}, nil)
	require.NoError(t, err)

	operator, err := client.New(user.Config(), client.Options{Scheme: scheme})
	require.NoError(t, err)

	// Each Role is bound to the operator user in a namespace of its own.
	roles := []struct {
		namespace string
		rules     func(t *testing.T) []rbacv1.PolicyRule
	}{
		{namespace: "config-rbac", rules: configRBACRules},
		{namespace: "chart-kubernetes", rules: chartRules("kubernetes")},
		{namespace: "chart-openshift", rules: chartRules("openshift")},
	}

	for _, role := range roles {
		t.Run(role.namespace, func(t *testing.T) {
			bindRole(t, admin, operator, role.namespace, role.rules(t))
			requireControllerReferences(t, admin, operator, scheme, role.namespace)
		})
	}
}

// requireControllerReferences writes, as the operator user, each controller reference the controllers set.
func requireControllerReferences(t *testing.T, admin, operator client.Client, scheme *runtime.Scheme, ns string) {
	t.Helper()

	pipeline := &cdPipeApi.CDPipeline{
		ObjectMeta: objectMeta(ns, pipelineName),
		Spec: cdPipeApi.CDPipelineSpec{
			Name:               pipelineName,
			DeploymentType:     "container",
			Applications:       []string{"app"},
			InputDockerStreams: []string{"app-main"},
		},
	}
	stage := newStage(ns, "app-dev")
	clusterSecret := &corev1.Secret{ObjectMeta: objectMeta(ns, "dev-cluster")}
	uncoveredOwner := &corev1.ConfigMap{ObjectMeta: objectMeta(ns, "uncovered-owner")}

	for _, owner := range []client.Object{pipeline, stage, clusterSecret, uncoveredOwner} {
		require.NoError(t, admin.Create(t.Context(), owner))
	}

	t.Run("rejects an owner without update on its finalizers", func(t *testing.T) {
		dependent := &corev1.ConfigMap{ObjectMeta: objectMeta(ns, "rejected")}
		require.NoError(t, controllerutil.SetControllerReference(uncoveredOwner, dependent, scheme))

		err := operator.Create(t.Context(), dependent)
		require.True(t, k8sErrors.IsForbidden(err), "want Forbidden, got %v", err)
		require.ErrorContains(t, err, "blockOwnerDeletion")
	})

	// One row per owner relation the operator writes with SetControllerReference;
	// a controller that starts writing a new one adds its row here.
	tests := []struct {
		name      string
		owner     client.Object
		dependent client.Object
		write     verb
	}{
		{
			name:      "CDPipeline controls an existing Stage",
			owner:     pipeline,
			dependent: newStage(ns, "app-qa"),
			write:     verbPatch,
		},
		{
			name:      "Stage controls a new ConfigMap",
			owner:     stage,
			dependent: &corev1.ConfigMap{ObjectMeta: objectMeta(ns, "app-dev")},
			write:     verbCreate,
		},
		{
			name:  "Stage controls a new CodebaseImageStream",
			owner: stage,
			dependent: &codebaseApi.CodebaseImageStream{
				ObjectMeta: objectMeta(ns, "app-dev-app-verified"),
				Spec:       codebaseApi.CodebaseImageStreamSpec{Codebase: "app", ImageName: "registry.example.com/app"},
			},
			write: verbCreate,
		},
		{
			name:      "Secret controls a new Secret",
			owner:     clusterSecret,
			dependent: &corev1.Secret{ObjectMeta: objectMeta(ns, "dev-cluster-argocd-cluster")},
			write:     verbCreate,
		},
		{
			name:      "Secret controls an existing Secret",
			owner:     clusterSecret,
			dependent: &corev1.Secret{ObjectMeta: objectMeta(ns, "dev")},
			write:     verbUpdate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.write != verbCreate {
				require.NoError(t, admin.Create(t.Context(), tt.dependent))
			}

			base, ok := tt.dependent.DeepCopyObject().(client.Object)
			require.True(t, ok)
			require.NoError(t, controllerutil.SetControllerReference(tt.owner, tt.dependent, scheme))

			switch tt.write {
			case verbCreate:
				require.NoError(t, operator.Create(t.Context(), tt.dependent))
			case verbUpdate:
				require.NoError(t, operator.Update(t.Context(), tt.dependent))
			case verbPatch:
				require.NoError(t, operator.Patch(t.Context(), tt.dependent, client.MergeFrom(base)))
			default:
				t.Fatalf("unknown verb %q", tt.write)
			}
		})
	}
}

// bindRole binds rules to the operator user in a new namespace.
func bindRole(t *testing.T, admin, operator client.Client, ns string, rules []rbacv1.PolicyRule) {
	t.Helper()

	require.NotEmpty(t, rules)
	require.NoError(t, admin.Create(t.Context(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	require.NoError(t, admin.Create(t.Context(), &rbacv1.Role{ObjectMeta: objectMeta(ns, operatorUser), Rules: rules}))
	require.NoError(t, admin.Create(t.Context(), &rbacv1.RoleBinding{
		ObjectMeta: objectMeta(ns, operatorUser),
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: operatorUser},
		Subjects:   []rbacv1.Subject{{APIGroup: rbacv1.GroupName, Kind: rbacv1.UserKind, Name: operatorUser}},
	}))

	require.Eventually(t, func() bool {
		return operator.List(t.Context(), &cdPipeApi.StageList{}, client.InNamespace(ns)) == nil
	}, 10*time.Second, 100*time.Millisecond, "the RBAC authorizer did not serve the binding")
}

// configRBACRules returns the rules of the namespaced Role that `make manifests` generates from the RBAC markers.
func configRBACRules(t *testing.T) []rbacv1.PolicyRule {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "config", "rbac", "role.yaml"))
	require.NoError(t, err)

	return roleRules(t, strings.Split(string(data), "\n---"), "manager-role")
}

// chartRules returns the rules of the chart's operator Role for platform. Every line holding a Helm
// expression is dropped, so the Role keeps the rules under every condition.
func chartRules(platform string) func(t *testing.T) []rbacv1.PolicyRule {
	return func(t *testing.T) []rbacv1.PolicyRule {
		t.Helper()

		data, err := os.ReadFile(filepath.Join("..", "..", "deploy-templates", "templates", "role_"+platform+".yaml"))
		require.NoError(t, err)

		var kept []string

		for _, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(line, "{{") {
				kept = append(kept, line)
			}
		}

		var d roleDoc
		require.NoError(t, yaml.Unmarshal([]byte(strings.Join(kept, "\n")), &d))
		require.Equal(t, "Role", d.Kind)

		return d.Rules
	}
}

func roleRules(t *testing.T, docs []string, name string) []rbacv1.PolicyRule {
	t.Helper()

	for _, raw := range docs {
		var d roleDoc
		if yaml.Unmarshal([]byte(raw), &d) != nil {
			continue
		}

		if d.Kind == "Role" && d.Metadata.Name == name {
			return d.Rules
		}
	}

	t.Fatalf("Role %s not found", name)

	return nil
}

func newStage(ns, name string) *cdPipeApi.Stage {
	return &cdPipeApi.Stage{
		ObjectMeta: objectMeta(ns, name),
		Spec: cdPipeApi.StageSpec{
			Name:         name,
			CdPipeline:   pipelineName,
			Description:  name,
			TriggerType:  cdPipeApi.TriggerTypeManual,
			Namespace:    ns + "-" + name,
			QualityGates: []cdPipeApi.QualityGate{{QualityGateType: "manual", StepName: "approve"}},
		},
	}
}

func objectMeta(ns, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: ns}
}
