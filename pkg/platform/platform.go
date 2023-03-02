package platform

import (
	"os"
	"strconv"
)

const (
	TypeEnv            = "PLATFORM_TYPE"
	Openshift          = "openshift"
	Kubernetes         = "kubernetes"
	KioskEnabledEnv    = "KIOSK_ENABLED"
	ManageNamespaceEnv = "MANAGE_NAMESPACE"
)

func GetPlatformTypeEnv() string {
	pt, ok := os.LookupEnv(TypeEnv)
	if ok {
		return pt
	}

	return Kubernetes
}

// IsKubernetes returns true if platform type is kubernetes.
func IsKubernetes() bool {
	return GetPlatformTypeEnv() == Kubernetes
}

// IsOpenshift returns true if platform type is openshift.
func IsOpenshift() bool {
	return GetPlatformTypeEnv() == Openshift
}

// KioskEnabled returns true if kiosk is enabled.
// It is enabled if the environment variable KIOSK_ENABLED is set to true.
func KioskEnabled() bool {
	enabled, ok := os.LookupEnv(KioskEnabledEnv)
	if !ok {
		return false
	}

	b, err := strconv.ParseBool(enabled)
	if err != nil {
		return false
	}

	return b
}

// ManageNamespace returns true if namespace should be managed by the operator.
// If the environment variable MANAGE_NAMESPACE is not set, it returns true.
func ManageNamespace() bool {
	enabled, ok := os.LookupEnv(ManageNamespaceEnv)
	if !ok {
		return true
	}

	b, err := strconv.ParseBool(enabled)
	if err != nil {
		return true
	}

	return b
}
