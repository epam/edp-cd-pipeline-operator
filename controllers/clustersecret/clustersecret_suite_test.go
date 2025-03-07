package clustersecret

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/aws"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/aws/mocks"
)

var (
	cfg           *rest.Config
	k8sClient     client.Client
	testEnv       *envtest.Environment
	ctx           context.Context
	cancel        context.CancelFunc
	rawKubeConfig []byte
)

func TestClusterSecret(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "ClusterSecret Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.30.0-%s-%s", goruntime.GOOS, goruntime.GOARCH)),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	rawKubeConfig, err = ConvertRestConfigToKubeConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	sc := scheme.Scheme
	Expect(pipelineApi.AddToScheme(sc)).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: sc})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             sc,
		MetricsBindAddress: "0",
	})
	Expect(err).ToNot(HaveOccurred())

	err = NewReconcileClusterSecret(k8sManager.GetClient(), newTokenGenMock(), checkClusterConnectionWithFakeHosts).
		SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func ConvertRestConfigToKubeConfig(cfg *rest.Config) ([]byte, error) {
	kubeConfig := clientcmdapi.NewConfig()

	cluster := clientcmdapi.NewCluster()
	cluster.Server = cfg.Host
	cluster.CertificateAuthorityData = cfg.TLSClientConfig.CAData
	cluster.InsecureSkipTLSVerify = cfg.TLSClientConfig.Insecure

	kubecontext := clientcmdapi.NewContext()
	kubecontext.Cluster = "default-cluster"
	kubecontext.AuthInfo = "default-user"

	authInfo := clientcmdapi.NewAuthInfo()
	authInfo.Token = cfg.BearerToken
	authInfo.ClientKeyData = cfg.TLSClientConfig.KeyData
	authInfo.ClientCertificateData = cfg.TLSClientConfig.CertData

	kubeConfig.Clusters["default-cluster"] = cluster
	kubeConfig.Contexts["default-context"] = kubecontext
	kubeConfig.AuthInfos["default-user"] = authInfo
	kubeConfig.CurrentContext = "default-context"

	rawConfig, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rest.Config to kubeconfig: %w", err)
	}

	return rawConfig, nil
}

func newTokenGenMock() *mocks.MockAIMAuthTokenGenerator {
	tokenGenMock := mocks.NewMockAIMAuthTokenGenerator(GinkgoT())
	tokenGenMock.On("GetWithRole", mock.Anything, mock.Anything).
		Return(func(clusterName, roleARN string) (aws.Token, error) {
			if clusterName == "cluster-error" {
				return aws.Token{}, errors.New("failed to generate token")
			}

			return aws.Token{
				Token:      "secret token",
				Expiration: time.Now().Add(15 * time.Minute),
			}, nil
		}).Maybe()

	return tokenGenMock
}

func checkClusterConnectionWithFakeHosts(ctx context.Context, restConf *rest.Config) error {
	if strings.Contains(restConf.Host, "fake-cluster-success") {
		return nil
	}

	if strings.Contains(restConf.Host, "fake-cluster-error") {
		return errors.New("failed to connect to cluster")
	}

	return CheckClusterConnection(ctx, restConf)
}
