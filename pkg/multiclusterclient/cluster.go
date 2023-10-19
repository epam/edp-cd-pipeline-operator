package multiclusterclient

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Constants from https://github.com/argoproj/argo-cd/blob/v2.8.4/pkg/apis/application/v1alpha1/cluster_constants.go#L45.
// k8sClientConfigQPS controls the QPS to be used in K8s REST client configs.
const k8sClientConfigQPS = 50

// Cluster is the definition of a cluster resource.
// It is based on the argocd Cluster struct because we use the same secret for external cluster authentication.
// See: https://github.com/argoproj/argo-cd/blob/v2.8.4/pkg/apis/application/v1alpha1/types.go#L1666.
type Cluster struct {
	// Server is the API server URL of the Kubernetes cluster
	Server string `json:"server" protobuf:"bytes,1,opt,name=server"`
	// Name of the cluster. If omitted, will use the server address
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
	// Config holds cluster information for connecting to a cluster
	Config ClusterConfig `json:"config" protobuf:"bytes,3,opt,name=config"`
}

// ClusterConfig is the configuration attributes. This structure is subset of the go-client
// rest.Config with annotations added for marshalling.
type ClusterConfig struct {
	// Server requires Basic authentication
	Username string `json:"username,omitempty" protobuf:"bytes,1,opt,name=username"`
	Password string `json:"password,omitempty" protobuf:"bytes,2,opt,name=password"`

	// Server requires Bearer authentication. This client will not attempt to use
	// refresh tokens for an OAuth2 flow.
	BearerToken string `json:"bearerToken,omitempty" protobuf:"bytes,3,opt,name=bearerToken"`

	// TLSClientConfig contains settings to enable transport layer security
	TLSClientConfig `json:"tlsClientConfig" protobuf:"bytes,4,opt,name=tlsClientConfig"`

	// ExecProviderConfig contains configuration for an exec provider
	ExecProviderConfig *ExecProviderConfig `json:"execProviderConfig,omitempty" protobuf:"bytes,6,opt,name=execProviderConfig"`
}

// TLSClientConfig contains settings to enable transport layer security.
type TLSClientConfig struct {
	// Insecure specifies that the server should be accessed without verifying the TLS certificate. For testing only.
	Insecure bool `json:"insecure" protobuf:"bytes,1,opt,name=insecure"`
	// ServerName is passed to the server for SNI and is used in the client to check server
	// certificates against. If ServerName is empty, the hostname used to contact the
	// server is used.
	ServerName string `json:"serverName,omitempty" protobuf:"bytes,2,opt,name=serverName"`
	// CertData holds PEM-encoded bytes (typically read from a client certificate file).
	// CertData takes precedence over CertFile
	CertData []byte `json:"certData,omitempty" protobuf:"bytes,3,opt,name=certData"`
	// KeyData holds PEM-encoded bytes (typically read from a client certificate key file).
	// KeyData takes precedence over KeyFile
	KeyData []byte `json:"keyData,omitempty" protobuf:"bytes,4,opt,name=keyData"`
	// CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
	// CAData takes precedence over CAFile
	CAData []byte `json:"caData,omitempty" protobuf:"bytes,5,opt,name=caData"`
}

// ExecProviderConfig is config used to call an external command to perform cluster authentication
// See: https://godoc.org/k8s.io/client-go/tools/clientcmd/api#ExecConfig
type ExecProviderConfig struct {
	// Command to execute
	Command string `json:"command,omitempty" protobuf:"bytes,1,opt,name=command"`

	// Arguments to pass to the command when executing it
	Args []string `json:"args,omitempty" protobuf:"bytes,2,rep,name=args"`

	// Env defines additional environment variables to expose to the process
	Env map[string]string `json:"env,omitempty" protobuf:"bytes,3,opt,name=env"`

	// Preferred input version of the ExecInfo
	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,4,opt,name=apiVersion"`

	// This text is shown to the user when the executable doesn't seem to be present
	InstallHint string `json:"installHint,omitempty" protobuf:"bytes,5,opt,name=installHint"`
}

// RestConfig returns a go-client REST config from cluster that might be serialized into the file using kube.WriteKubeConfig method.
func (c *Cluster) RestConfig() *rest.Config {
	var config *rest.Config

	tlsClientConfig := rest.TLSClientConfig{
		Insecure:   c.Config.TLSClientConfig.Insecure,
		ServerName: c.Config.TLSClientConfig.ServerName,
		CertData:   c.Config.TLSClientConfig.CertData,
		KeyData:    c.Config.TLSClientConfig.KeyData,
		CAData:     c.Config.TLSClientConfig.CAData,
	}

	switch {
	case c.Config.ExecProviderConfig != nil:
		var env []api.ExecEnvVar

		if c.Config.ExecProviderConfig.Env != nil {
			for key, value := range c.Config.ExecProviderConfig.Env {
				env = append(env, api.ExecEnvVar{
					Name:  key,
					Value: value,
				})
			}
		}

		config = &rest.Config{
			Host:            c.Server,
			TLSClientConfig: tlsClientConfig,
			ExecProvider: &api.ExecConfig{
				APIVersion:      c.Config.ExecProviderConfig.APIVersion,
				Command:         c.Config.ExecProviderConfig.Command,
				Args:            c.Config.ExecProviderConfig.Args,
				Env:             env,
				InstallHint:     c.Config.ExecProviderConfig.InstallHint,
				InteractiveMode: api.NeverExecInteractiveMode,
			},
		}
	default:
		config = &rest.Config{
			Host:            c.Server,
			Username:        c.Config.Username,
			Password:        c.Config.Password,
			BearerToken:     c.Config.BearerToken,
			TLSClientConfig: tlsClientConfig,
		}
	}

	config.QPS = k8sClientConfigQPS
	config.Burst = int(config.QPS * 2)

	return config
}
