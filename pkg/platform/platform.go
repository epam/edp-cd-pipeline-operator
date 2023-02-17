package platform

import "os"

const (
	Type       = "PLATFORM_TYPE"
	Openshift  = "openshift"
	Kubernetes = "kubernetes"
)

func GetPlatformTypeEnv() string {
	pt, ok := os.LookupEnv(Type)
	if ok {
		return pt
	}

	return Openshift
}

// IsKubernetes returns true if platform type is kubernetes.
func IsKubernetes() bool {
	return GetPlatformTypeEnv() == Kubernetes
}

// IsOpenshift returns true if platform type is openshift.
func IsOpenshift() bool {
	return GetPlatformTypeEnv() == Openshift
}
