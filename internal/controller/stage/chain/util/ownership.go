package util

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

var ErrNamespaceNotOwned = errors.New("namespace is not owned by the tenant")

// CheckNamespaceOwnership returns nil only if the namespace carries the tenant label
// with the given non-empty tenant value.
func CheckNamespaceOwnership(ns *corev1.Namespace, tenant string) error {
	if v := ns.Labels[TenantLabelName]; v != "" && v == tenant {
		return nil
	}

	return fmt.Errorf(
		"%w: namespace %s exists but is not owned by tenant %s: label %s must be %s",
		ErrNamespaceNotOwned, ns.Name, tenant, TenantLabelName, tenant,
	)
}

// OwnershipEnforced reports whether the target namespace ownership must be verified.
// It is false on OpenShift, where the Namespace object exists next to the Project and
// ownership is not verified yet, and for an in-cluster Stage when Capsule manages tenancy.
func OwnershipEnforced(stage *cdPipeApi.Stage) bool {
	if platform.IsOpenshift() {
		return false
	}

	inCluster := stage.Spec.ClusterName == "" || stage.Spec.ClusterName == cdPipeApi.InCluster

	return !platform.CapsuleEnabled() || !inCluster
}
