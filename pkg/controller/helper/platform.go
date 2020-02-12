package helper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform_util")

func GetOwnerReference(ownerKind string, ors []metav1.OwnerReference) *metav1.OwnerReference {
	log.V(2).Info("finding owner", "kind", ownerKind)
	if len(ors) == 0 {
		return nil
	}
	for _, o := range ors {
		if o.Kind == ownerKind {
			return &o
		}
	}
	return nil
}
