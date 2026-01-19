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
								//nolint:lll // Contains base64-encoded certificate data in JSON
								"config": []byte(`{
								  "apiVersion": "v1",
								  "kind": "Config",
								  "current-context": "default-context",
								  "preferences": {},
								  "clusters": [
									{
									  "cluster": {
										"server": "https://test-cluster",
										"certificate-authority-data": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURqVENDQW5XZ0F3SUJBZ0lFZG1uSWtqQU5CZ2txaGtpRzl3MEJBUXNGQURCYk1TY3dKUVlEVlFRRERCNVNaV2RsY25rZ1UyVnMKWmkxVGFXZHVaV1FnUTJWeWRHbG1hV05oZEdVeEl6QWhCZ05WQkFvTUdsSmxaMlZ5ZVN3Z2FIUjBjSE02THk5eVpXZGxjbmt1WTI5dApNUXN3Q1FZRFZRUUdFd0pWUVRBZ0Z3MHlOakF4TWpFd01EQXdNREJhR0E4eU1USTJNREV5TVRBNE16ZzFNMW93VHpFYk1Ca0dBMVVFCkF3d1NhSFIwY0hNNmRHVnpkQzFqYkhWemRHVnlNU013SVFZRFZRUUtEQnBTWldkbGNua3NJR2gwZEhCek9pOHZjbVZuWlhKNUxtTnYKYlRFTE1Ba0dBMVVFQmhNQ1ZVRXdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDTjhvTWw5MU40Q3FvSAo3R0J4Z2Rza3VWVGlXeXVrUVlRS3BtbktyMkRUUWRITnlIS1hlcWxMMjNITFVPUVNqZFRmb1JSOFlOdWc1NHRNdlZJM3F1YjQyK2gxCmVPekVxYk5rYkZNREdHOEk1QW9EejlGRFdTbUdWenZ3TlMyK1pvamFNNDdTRGlZeHdabldpTEFEY2x4M1o4SUZnMHRueXZXNjRrUDIKTDJxSWtYcjI2VVBmdUJMM2srczdwZjVEbXR5L3BzSFMvTFlVT1pGKy9zQ0ptRGpZT3U2OWl2eDJXcUdiQmU3aDJOSG5ua0NYYTh5bQoxRjl5cENSY3lmdCtJQmdYSmFodnF2T25RUzV4MkFRVXg1VzdMdGZIaXVMdUVrQVNuMTU2Q1dIYUlRN2xnaW5lYmFPSVN5bVhsTmpBCml1NEd1dkVNV0V2YlI2SUtUcWUvS1NVakFnTUJBQUdqWXpCaE1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0RnWURWUjBQQVFIL0JBUUQKQWdHR01CMEdBMVVkRGdRV0JCU24zLzRKSUgxOW53SjZxSGZBYjA0bW8xZ3J5akFmQmdOVkhTTUVHREFXZ0JTbjMvNEpJSDE5bndKNgpxSGZBYjA0bW8xZ3J5akFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBWXR1S3pPeDJvYXN6WVl6ZU9GalViSVBSVVkwZzJzS29jcitxCjA2SkNlcHY3MjNuQlAvNEo5MVBLVUJ6ZkZ2ZDczMUlNTWlQcm5GTWZsRzc0QlE1aUtXL1JUOTlQd3Fxcnl3Vzd5ZW5ZWlFQVFRjb24KODZJcXZYK096U01KUnRWU1d5bzhaUkIyb3hrRGpveXgyd2hWWFhUREVESENnd1g5OFdOR3dBMXUxK0FlTkwyUXJhQ1NrMlIvdWNIVgovc0t6cy9kaFRLZklQZnFGbVkxQUFRQzhyZG5Cb1FGU1dDN0pUdkVVSldhbktNOHBiRnNucUtZc3AwcmJQdzBlSjFqc2o2RkZuN2ViCm4yalNzQkZwNllpcEpNTU5wOUZVNDNSb1EybENjdS92M3ByY0JDakRGNWJWb3drSGluU1ZnZHlGT2YrSldWVW9POW5ETjZmMnBaWm0KeXc9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t"
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
		t.Run(tt.name, func(t *testing.T) {
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
