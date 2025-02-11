package clustersecret

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/argocd"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/aws"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/aws/mocks"
)

var (
	clusterConf = []byte(`{
	  "apiVersion": "v1",
	  "kind": "Config",
	  "current-context": "default-context",
	  "preferences": {},
	  "clusters": [
		{
		  "cluster": {
			"server": "https://test-cluster",
			"certificate-authority-data": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNVVENDQWZ1Z0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRUUZBREJYTVFzd0NRWURWUVFHRXdKRFRqRUwKTUFrR0ExVUVDQk1DVUU0eEN6QUpCZ05WQkFjVEFrTk9NUXN3Q1FZRFZRUUtFd0pQVGpFTE1Ba0dBMVVFQ3hNQwpWVTR4RkRBU0JnTlZCQU1UQzBobGNtOXVaeUJaWVc1bk1CNFhEVEExTURjeE5USXhNVGswTjFvWERUQTFNRGd4Ck5ESXhNVGswTjFvd1Z6RUxNQWtHQTFVRUJoTUNRMDR4Q3pBSkJnTlZCQWdUQWxCT01Rc3dDUVlEVlFRSEV3SkQKVGpFTE1Ba0dBMVVFQ2hNQ1QwNHhDekFKQmdOVkJBc1RBbFZPTVJRd0VnWURWUVFERXd0SVpYSnZibWNnV1dGdQpaekJjTUEwR0NTcUdTSWIzRFFFQkFRVUFBMHNBTUVnQ1FRQ3A1aG5HN29nQmh0bHlucE9TMjFjQmV3S0UvQjdqClYxNHFleXNsbnIyNnhaVXNTVmtvMzZabmhpYU8vemJNT29SY0tLOXZFY2dNdGNMRnVRVFdEbDNSQWdNQkFBR2oKZ2JFd2dhNHdIUVlEVlIwT0JCWUVGRlhJNzBrclhlUUR4WmdiYUNRb1I0alVEbmNFTUg4R0ExVWRJd1I0TUhhQQpGRlhJNzBrclhlUUR4WmdiYUNRb1I0alVEbmNFb1Z1a1dUQlhNUXN3Q1FZRFZRUUdFd0pEVGpFTE1Ba0dBMVVFCkNCTUNVRTR4Q3pBSkJnTlZCQWNUQWtOT01Rc3dDUVlEVlFRS0V3SlBUakVMTUFrR0ExVUVDeE1DVlU0eEZEQVMKQmdOVkJBTVRDMGhsY205dVp5QlpZVzVuZ2dFQU1Bd0dBMVVkRXdRRk1BTUJBZjh3RFFZSktvWklodmNOQVFFRQpCUUFEUVFBL3VnekJyampLOWpjV25EVmZHSGxrM2ljTlJxMG9WN1JpMzJ6LytIUVg2N2FSZmdadTdLV2RJK0p1CldtN0RDZnJQTkdWd0ZXVVFPbXNQdWU5clpCZ08KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
		  },
		  "name": "default-cluster"
		}
	  ],
	  "contexts": [
		{
		  "context": {
			"cluster": "default-cluster",
			"user": "default-user"
		  },
		  "name": "default-context"
		}
	  ],
	  "users": [
		{
		  "user": {
			"token": "token-123"
		  },
		  "name": "default-user"
		}
	  ]
	}`)

	irsa = []byte(`{
      "awsAuthConfig": {
        "clusterName": "my-eks-cluster-name",
        "roleARN": "arn:aws:iam::<AWS_ACCOUNT_ID>:role/<IAM_ROLE_NAME>"
      },
      "tlsClientConfig": {
        "insecure": false,
        "caData": "Y2EgZGF0YQ=="
      }        
    }`)
)

func TestReconcileClusterSecret_Reconcile(t *testing.T) {
	t.Parallel()

	mockTokenGen := func(t *testing.T) aws.AIMAuthTokenGenerator {
		return mocks.NewMockAIMAuthTokenGenerator(t)
	}
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name     string
		client   func(t *testing.T) client.Client
		tokenGen func(t *testing.T) aws.AIMAuthTokenGenerator
		wantErr  require.ErrorAssertionFunc
		want     func(t *testing.T, cl client.Client)
	}{
		{
			name: "should create argocd cluster secret",
			client: func(t *testing.T) client.Client {
				s := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							integrationSecretTypeLabel: kubeconfClusterIntegrationSecret,
						},
					},
					Data: map[string][]byte{
						"config": clusterConf,
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(s).Build()
			},
			tokenGen: mockTokenGen,
			wantErr:  require.NoError,
			want: func(t *testing.T, cl client.Client) {
				secret := &corev1.Secret{}
				require.NoError(t, cl.Get(context.Background(), client.ObjectKey{
					Name:      "test-argocd-cluster",
					Namespace: "default",
				}, secret))

				require.Contains(t, secret.Data, "name")
				require.Contains(t, secret.Data, "server")
				require.Contains(t, secret.Data, "config")
				require.Contains(t, secret.GetLabels(), argocd.ClusterLabel)
			},
		},
		{
			name: "should update argocd cluster secret",
			client: func(t *testing.T) client.Client {
				s := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							integrationSecretTypeLabel: kubeconfClusterIntegrationSecret,
						},
					},
					Data: map[string][]byte{
						"config": clusterConf,
					},
				}

				argoSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-argocd-cluster",
						Namespace: "default",
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(s, argoSecret).Build()
			},
			tokenGen: mockTokenGen,
			wantErr:  require.NoError,
			want: func(t *testing.T, cl client.Client) {
				secret := &corev1.Secret{}
				require.NoError(t, cl.Get(context.Background(), client.ObjectKey{
					Name:      "test-argocd-cluster",
					Namespace: "default",
				}, secret))

				require.Contains(t, secret.Data, "name")
				require.Contains(t, secret.Data, "server")
				require.Contains(t, secret.Data, "config")
				require.Contains(t, secret.GetLabels(), argocd.ClusterLabel)
			},
		},
		{
			name: "invalid cluster secret config",
			client: func(t *testing.T) client.Client {
				s := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							integrationSecretTypeLabel: kubeconfClusterIntegrationSecret,
						},
					},
					Data: map[string][]byte{
						"config": []byte(`not json`),
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(s).Build()
			},
			tokenGen: mockTokenGen,
			wantErr:  require.Error,
			want:     func(t *testing.T, cl client.Client) {},
		},
		{
			name: "irsa cluster secret",
			client: func(t *testing.T) client.Client {
				s := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							integrationSecretTypeLabel: irsaClusterIntegrationSecret,
							argocd.ClusterLabel:        argocd.ClusterLabelVal,
						},
					},
					Data: map[string][]byte{
						"server": []byte("https://test-cluster"),
						"config": irsa,
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(s).Build()
			},
			tokenGen: func(t *testing.T) aws.AIMAuthTokenGenerator {
				m := mocks.NewMockAIMAuthTokenGenerator(t)

				m.On("GetWithRole", "my-eks-cluster-name", "arn:aws:iam::<AWS_ACCOUNT_ID>:role/<IAM_ROLE_NAME>").
					Return(aws.Token{
						Token:      "some-token",
						Expiration: time.Now().Add(15 * time.Minute),
					}, nil)

				return m
			},
			wantErr: require.NoError,
			want: func(t *testing.T, cl client.Client) {
				secret := &corev1.Secret{}
				require.NoError(t, cl.Get(context.Background(), client.ObjectKey{
					Name:      "test-cluster",
					Namespace: "default",
				}, secret))

				require.Contains(t, secret.Data, "config")
				require.Contains(t, secret.GetLabels(), integrationSecretTypeLabel)
				require.Equal(t, kubeconfClusterIntegrationSecret, secret.GetLabels()[integrationSecretTypeLabel])
				require.Equal(t, "false", secret.GetLabels()[generateArgoCDSecretLabel])
			},
		},
		{
			name: "invalid irsa cluster data",
			client: func(t *testing.T) client.Client {
				s := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Labels: map[string]string{
							integrationSecretTypeLabel: irsaClusterIntegrationSecret,
							argocd.ClusterLabel:        argocd.ClusterLabelVal,
						},
					},
					Data: map[string][]byte{
						"server": []byte("https://test-cluster"),
						"config": []byte("not json"),
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(s).Build()
			},
			tokenGen: func(t *testing.T) aws.AIMAuthTokenGenerator {
				return mocks.NewMockAIMAuthTokenGenerator(t)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to generate kubeconfig")
			},
			want: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "cluster secret not found",
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			tokenGen: mockTokenGen,
			wantErr:  require.NoError,
			want:     func(t *testing.T, cl client.Client) {},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewReconcileClusterSecret(tt.client(t), tt.tokenGen(t))
			_, err := r.Reconcile(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				reconcile.Request{
					NamespacedName: client.ObjectKey{
						Name:      "test",
						Namespace: "default",
					},
				})

			tt.wantErr(t, err)
		})
	}
}

func Test_needToReconcile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		object client.Object
		want   bool
	}{
		{
			name: "kube conf to argoCD secret",
			object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						integrationSecretTypeLabel: kubeconfClusterIntegrationSecret,
					},
				},
			},
			want: true,
		},
		{
			name: "kube conf to argoCD secret with skip label",
			object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						integrationSecretTypeLabel: kubeconfClusterIntegrationSecret,
						generateArgoCDSecretLabel:  "false",
					},
				},
			},
			want: false,
		},
		{
			name: "argoCD IRSA secret to kube conf",
			object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						integrationSecretTypeLabel: irsaClusterIntegrationSecret,
					},
				},
			},
			want: true,
		},
		{
			name:   "should return false",
			object: &corev1.Secret{},
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := needToReconcile(tt.object)
			require.Equal(t, tt.want, got)
		})
	}
}
