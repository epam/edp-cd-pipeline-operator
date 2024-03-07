package multiclusterclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

func TestClientProvider_GetClusterClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		clusterName           string
		internalClusterClient func(t *testing.T) client.Client
		want                  require.ValueAssertionFunc
		wantErr               require.ErrorAssertionFunc
	}{
		{
			name:        "should return external cluster client",
			clusterName: "external-cluster",
			internalClusterClient: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "external-cluster",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"config": []byte(`{
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
								}`,
								),
							},
						},
					).
					Build()
			},
			want:    require.NotNil,
			wantErr: require.NoError,
		},
		{
			name:        "invalid secret data",
			clusterName: "external-cluster",
			internalClusterClient: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "external-cluster",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"config": []byte(`not json data`),
							},
						},
					).
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create rest config from cluster secret")
			},
		},
		{
			name:        "secret does not contain config data",
			clusterName: "external-cluster",
			internalClusterClient: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "external-cluster",
								Namespace: "default",
							},
							Data: map[string][]byte{},
						},
					).
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no config data in the secret")
			},
		},
		{
			name:        "secret not found",
			clusterName: "external-cluster",
			internalClusterClient: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects().
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "not found")
			},
		},
		{
			name:        "in-cluster",
			clusterName: cdPipeApi.InCluster,
			internalClusterClient: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects().
					Build()
			},
			want:    require.NotNil,
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := NewClientProvider(tt.internalClusterClient(t))
			got, err := c.GetClusterClient(
				context.Background(),
				"default",
				tt.clusterName,
				client.Options{
					Mapper: meta.NewDefaultRESTMapper([]schema.GroupVersion{}),
				},
			)

			tt.wantErr(t, err)
			tt.want(t, got)
		})
	}
}
