package helper

import "os"

const (
	platformType       = "PLATFORM_TYPE"
	PlatformOpenshift  = "openshift"
	PlatformKubernetes = "kubernetes"
)

func GetPlatformTypeEnv() string {
	pt, ok := os.LookupEnv(platformType)
	if ok {
		return pt
	}
	return PlatformOpenshift
}
