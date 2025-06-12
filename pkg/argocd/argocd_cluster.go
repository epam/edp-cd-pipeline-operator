package argocd

import (
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/aws"
)

const (
	ClusterLabel    = "argocd.argoproj.io/secret-type"
	ClusterLabelVal = "cluster"
)

// ClusterConfig is the ArgoCD cluster configuration.
type ClusterConfig struct {
	// BearerToken is the token to authenticate with the cluster.
	BearerToken string `json:"bearerToken,omitempty"`

	// TLSClientConfig contains settings to enable transport layer security
	TLSClientConfig `json:"tlsClientConfig"`
}

// TLSClientConfig contains settings to enable transport layer security.
type TLSClientConfig struct {
	// Insecure specifies that the server should be accessed without verifying the TLS certificate. For testing only.
	Insecure bool `json:"insecure"`

	// CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
	// CAData takes precedence over CAFile
	CAData []byte `json:"caData,omitempty"`
}

type IrsaClusterConfig struct {
	// AwsAuthConfig contains settings to authenticate with the cluster using AWS IAM Role.
	AwsAuthConfig AwsAuthConfig `json:"awsAuthConfig"`

	// TLSClientConfig contains settings to enable transport layer security
	TLSClientConfig TLSClientConfig `json:"tlsClientConfig"`
}

type AwsAuthConfig struct {
	ClusterName string `json:"clusterName"`
	RoleARN     string `json:"roleARN"`
}

func AddClusterLabel(argoClusterSecret *corev1.Secret) {
	labels := argoClusterSecret.GetLabels()
	if labels == nil {
		labels = make(map[string]string, 1)
	}

	labels[ClusterLabel] = ClusterLabelVal

	argoClusterSecret.SetLabels(labels)
}

// SecretToIRSACluster converts the ArgoCD IRSA cluster secret config to IrsaClusterConfig.
// Secret is in format: https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#eks
func SecretToIRSACluster(secret *corev1.Secret) (*IrsaClusterConfig, error) {
	c := &IrsaClusterConfig{}

	if err := json.Unmarshal(secret.Data["config"], c); err != nil {
		return c, fmt.Errorf("failed to unmarshal cluster config: %w", err)
	}

	return c, nil
}

// ArgoIRSAClusterSecretToKubeconfig converts the ArgoCD IRSA cluster secret to kubeconfig format.
// For authentication, it uses the AWS IAM Role and generates a token.
// Be aware that the token is valid for 15 minutes.
func ArgoIRSAClusterSecretToKubeconfig(secret *corev1.Secret, tokenGenerator aws.AIMAuthTokenGenerator) ([]byte, error) {
	argoConf, err := SecretToIRSACluster(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get IRSA secret config: %w", err)
	}

	tk, err := tokenGenerator.GetWithRole(argoConf.AwsAuthConfig.ClusterName, argoConf.AwsAuthConfig.RoleARN)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	config := clientcmdapi.NewConfig()

	cluster := clientcmdapi.NewCluster()
	cluster.Server = string(secret.Data["server"])
	cluster.CertificateAuthorityData = argoConf.TLSClientConfig.CAData
	cluster.InsecureSkipTLSVerify = argoConf.TLSClientConfig.Insecure

	clusterContext := clientcmdapi.NewContext()
	clusterContext.Cluster = argoConf.AwsAuthConfig.ClusterName
	clusterContext.AuthInfo = "default-user"

	authInfo := clientcmdapi.NewAuthInfo()
	authInfo.Token = tk.Token

	config.Clusters[argoConf.AwsAuthConfig.ClusterName] = cluster
	config.Contexts[argoConf.AwsAuthConfig.ClusterName] = clusterContext
	config.AuthInfos["default-user"] = authInfo
	config.CurrentContext = argoConf.AwsAuthConfig.ClusterName
	config.Extensions["irsa"] = newTokenExpirationClusterConfigExtension(tk.Expiration)

	raw, err := clientcmd.Write(*config)
	if err != nil {
		return nil, fmt.Errorf("failed to conver kubeconfig: %w", err)
	}

	return raw, nil
}

func newTokenExpirationClusterConfigExtension(tokenExpiration time.Time) *runtime.Unknown {
	return &runtime.Unknown{
		Raw: []byte(fmt.Sprintf(`{"tokenExpiration":%q}`, tokenExpiration.Format(time.RFC3339))),
	}
}
