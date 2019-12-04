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

func IsOpenshift() bool {
	return GetPlatformTypeEnv() == PlatformOpenshift
}

type PipelineStage struct {
	Name string `json:"name"`
	StepName string `json:"step_name"`
}