package chart

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

const vapName = "edp-cd-pipeline-operator-krci-tenant-scope"

type vapDoc struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Variables []struct {
			Name       string `json:"name"`
			Expression string `json:"expression"`
		} `json:"variables"`
	} `json:"spec"`
}

func renderVAP(t *testing.T, sets ...string) *vapDoc {
	t.Helper()

	args := make([]string, 0, 7+2*len(sets))
	args = append(args, "template", "x", "../../deploy-templates", "--namespace", "krci", "--set", "global.platform=kubernetes")

	for _, s := range sets {
		args = append(args, "--set", s)
	}

	out, err := exec.Command("helm", args...).CombinedOutput()
	require.NoError(t, err, string(out))

	for _, raw := range strings.Split(string(out), "\n---") {
		var d vapDoc
		if yaml.Unmarshal([]byte(raw), &d) != nil {
			continue
		}

		if d.Kind == "ValidatingAdmissionPolicy" && d.Metadata.Name == vapName {
			return &d
		}
	}

	return nil
}

func (d *vapDoc) variable(t *testing.T, name string) string {
	t.Helper()

	for _, v := range d.Spec.Variables {
		if v.Name == name {
			return v.Expression
		}
	}

	t.Fatalf("variable %q not found", name)

	return ""
}

func TestTenantScopeVAPRender(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not in PATH")
	}

	for _, manage := range []string{"true", "false"} {
		d := renderVAP(t, "tenancyEngine=none", "manageNamespace="+manage)
		require.NotNil(t, d, "manageNamespace=%s", manage)
		scoped := d.variable(t, "tenantScoped")
		require.Contains(t, scoped, "oldObject.metadata.labels['app.edp.epam.com/tenant'] == 'krci'")
		require.Contains(t, scoped, "namespaceObject.metadata.labels['app.edp.epam.com/tenant'] == 'krci'")

		own := d.variable(t, "isOwnNamespace")
		require.Contains(t, own, "request.namespace == 'krci'")
		require.Contains(t, own, "== 'secrets'")
		require.Contains(t, own, "== 'rolebindings'")
		require.Contains(t, own, "startsWith('eso-')")
		require.NotContains(t, own, "serviceaccounts")
	}

	require.Nil(t, renderVAP(t, "tenancyEngine=capsule"))
	require.Nil(t, renderVAP(t, "tenancyEngine=none", "tenantScopePolicy.enabled=false"))
}
