package platform

import (
	"os"
	"strconv"
)

const (
	TypeEnv                = "PLATFORM_TYPE"
	TenancyEngineEnv       = "TENANCY_ENGINE"
	ManageNamespaceEnv     = "MANAGE_NAMESPACE"
	Openshift              = "openshift"
	Kubernetes             = "kubernetes"
	TenancyEngineCapsule   = "capsule"
	OIDCAdminGroupName     = "OIDC_ADMIN_GROUP_NAME"
	OIDCDeveloperGroupName = "OIDC_DEVELOPER_GROUP_NAME"
	SecretManagerEnv       = "SECRET_MANAGER"
	SecretManagerESO       = "eso"
	SecretManagerOwn       = "own"
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

// CapsuleEnabled returns true if capsule is enabled.
func CapsuleEnabled() bool {
	return os.Getenv(TenancyEngineEnv) == TenancyEngineCapsule
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

// GetSecretManager returns the secret manager mode: eso, own, or empty/none when disabled.
func GetSecretManager() string {
	return os.Getenv(SecretManagerEnv)
}

// IsSecretManagerESO returns true if secrets are managed by the External Secrets Operator.
func IsSecretManagerESO() bool {
	return GetSecretManager() == SecretManagerESO
}

// GetOIDCAdminGroupName returns the name of the OIDC admin group.
func GetOIDCAdminGroupName() string {
	return os.Getenv(OIDCAdminGroupName)
}

// GetOIDCDeveloperGroupName returns the name of the OIDC developer group.
func GetOIDCDeveloperGroupName() string {
	return os.Getenv(OIDCDeveloperGroupName)
}
