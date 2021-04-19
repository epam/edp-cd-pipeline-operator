package helper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("platform_util")

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
