package util

import (
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

// RuntimeSecurityRbName is the name of the per-Stage-namespace RoleBinding.
// The chart tests assert that the delete rules of the operator ClusterRole and of the admission policy carry it.
const RuntimeSecurityRbName = "runtime-security-reader"

// RuntimeSecurityApplies reports whether the runtime security RoleBinding applies to the Stage:
// the feature is on, the Stage is in-cluster and the platform verifies namespace ownership.
// OpenShift does not verify ownership, so no read is granted there.
func RuntimeSecurityApplies(stage *cdPipeApi.Stage) bool {
	return platform.SecurityEnabled() && stage.InCluster() && !platform.IsOpenshift()
}
