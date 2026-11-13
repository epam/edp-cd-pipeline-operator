package chart

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
)

const (
	securityRoleName   = "edp-cd-pipeline-operator-krci-runtime-security"
	securityBinderName = securityRoleName + "-binder"
)

type securityDoc struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	RoleRef struct {
		Name string `json:"name"`
	} `json:"roleRef"`
	Rules []struct {
		APIGroups     []string `json:"apiGroups"`
		Resources     []string `json:"resources"`
		ResourceNames []string `json:"resourceNames"`
		Verbs         []string `json:"verbs"`
	} `json:"rules"`
	Spec struct {
		Template struct {
			Spec struct {
				Containers []struct {
					Env []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"env"`
				} `json:"containers"`
			} `json:"spec"`
		} `json:"template"`
	} `json:"spec"`
}

func renderSecurityDocs(t *testing.T, sets ...string) []securityDoc {
	t.Helper()

	var docs []securityDoc

	for _, raw := range strings.Split(helmTemplate(t, sets...), "\n---") {
		var d securityDoc
		if yaml.Unmarshal([]byte(raw), &d) == nil {
			docs = append(docs, d)
		}
	}

	return docs
}

func findDoc(docs []securityDoc, kind, name string) *securityDoc {
	for i := range docs {
		if docs[i].Kind == kind && docs[i].Metadata.Name == name {
			return &docs[i]
		}
	}

	return nil
}

func securityEnv(docs []securityDoc) map[string]string {
	env := map[string]string{}

	d := findDoc(docs, "Deployment", "cd-pipeline-operator")
	if d == nil {
		return env
	}

	for _, e := range d.Spec.Template.Spec.Containers[0].Env {
		if strings.HasPrefix(e.Name, "SECURITY_") {
			env[e.Name] = e.Value
		}
	}

	return env
}

func TestRuntimeSecurityRBACRender(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not in PATH")
	}

	for _, tenancy := range []string{"none", "capsule"} {
		t.Run(tenancy, func(t *testing.T) {
			docs := renderSecurityDocs(t, "security.enabled=true", "tenancyEngine="+tenancy)

			role := findDoc(docs, "ClusterRole", securityRoleName)
			require.NotNil(t, role)
			require.Len(t, role.Rules, 3)
			require.Equal(t, []string{"aquasecurity.github.io"}, role.Rules[0].APIGroups)
			require.ElementsMatch(t, []string{"vulnerabilityreports", "exposedsecretreports"}, role.Rules[0].Resources)
			require.ElementsMatch(t, []string{"get", "list"}, role.Rules[0].Verbs)
			require.Equal(t, []string{"apps"}, role.Rules[1].APIGroups)
			require.ElementsMatch(t, []string{"replicasets", "deployments"}, role.Rules[1].Resources)
			require.ElementsMatch(t, []string{"get", "list"}, role.Rules[1].Verbs)
			require.Equal(t, []string{"batch"}, role.Rules[2].APIGroups)
			require.Equal(t, []string{"jobs"}, role.Rules[2].Resources)
			require.ElementsMatch(t, []string{"get", "list"}, role.Rules[2].Verbs)

			// The reads are never bound cluster-wide.
			require.Nil(t, findDoc(docs, "ClusterRoleBinding", securityRoleName))

			for _, d := range docs {
				if d.Kind == "ClusterRoleBinding" {
					require.NotEqual(t, securityRoleName, d.RoleRef.Name)
				}
			}

			binder := findDoc(docs, "ClusterRole", securityBinderName)
			require.NotNil(t, binder)
			require.Equal(t, []string{"clusterroles"}, binder.Rules[0].Resources)
			require.Equal(t, []string{securityRoleName}, binder.Rules[0].ResourceNames)
			require.Equal(t, []string{"bind"}, binder.Rules[0].Verbs)

			// Only the binder is bound, and it holds nothing but bind and the delete of one named RoleBinding.
			crb := findDoc(docs, "ClusterRoleBinding", securityBinderName)
			require.NotNil(t, crb)
			require.Equal(t, securityBinderName, crb.RoleRef.Name)

			if tenancy == "none" {
				require.Len(t, binder.Rules, 2)
				require.Equal(t, []string{"rolebindings"}, binder.Rules[1].Resources)
				require.Equal(t, []string{util.RuntimeSecurityRbName}, binder.Rules[1].ResourceNames)
				require.Equal(t, []string{"delete"}, binder.Rules[1].Verbs)
			} else {
				require.Len(t, binder.Rules, 1)
			}
		})
	}
}

func TestRuntimeSecurityRBACIsNotRendered(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not in PATH")
	}

	tests := []struct {
		name string
		sets []string
	}{
		{name: "disabled by default"},
		{name: "disabled explicitly", sets: []string{"security.enabled=false"}},
		{name: "openshift", sets: []string{"security.enabled=true", "global.platform=openshift"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := renderSecurityDocs(t, tt.sets...)

			require.Nil(t, findDoc(docs, "ClusterRole", securityRoleName))
			require.Nil(t, findDoc(docs, "ClusterRole", securityBinderName))
			require.Nil(t, findDoc(docs, "ClusterRoleBinding", securityRoleName))
			require.Nil(t, findDoc(docs, "ClusterRoleBinding", securityBinderName))
			require.Empty(t, securityEnv(docs))
		})
	}
}

func TestRuntimeSecurityEnv(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not in PATH")
	}

	require.Empty(t, securityEnv(renderSecurityDocs(t)))

	require.Equal(t, map[string]string{
		"SECURITY_ENABLED":             "true",
		"SECURITY_SERVICE_ACCOUNT":     "edp-cd-pipeline-operator",
		"SECURITY_READER_CLUSTER_ROLE": securityRoleName,
		"SECURITY_SECRET_NAME":         "ci-defectdojo",
		"SECURITY_PRODUCT_TYPE":        "KubeRocketCI",
		"SECURITY_POLL_INTERVAL":       "5m",
		"SECURITY_CONCURRENCY":         "2",
	}, securityEnv(renderSecurityDocs(t, "security.enabled=true")))

	require.Equal(t, map[string]string{
		"SECURITY_ENABLED":             "true",
		"SECURITY_SERVICE_ACCOUNT":     "edp-cd-pipeline-operator",
		"SECURITY_READER_CLUSTER_ROLE": securityRoleName,
		"SECURITY_SECRET_NAME":         "dd",
		"SECURITY_PRODUCT_TYPE":        "Platform",
		"SECURITY_POLL_INTERVAL":       "90s",
		"SECURITY_CONCURRENCY":         "4",
	}, securityEnv(renderSecurityDocs(t,
		"security.enabled=true", "security.secretName=dd", "security.productType=Platform",
		"security.pollInterval=90s", "security.concurrency=4")))
}

// The ESO delete policy must let the operator delete the runtime security RoleBinding of a Stage namespace,
// otherwise the delete chain is denied there and Stage deletion stalls.
func TestEsoDeletePolicyAllowsRuntimeSecurityRoleBinding(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not in PATH")
	}

	out := helmTemplate(t, "secretManager=eso", "tenancyEngine=none")
	require.Contains(t, out, "request.name == '"+util.RuntimeSecurityRbName+"'")
}
