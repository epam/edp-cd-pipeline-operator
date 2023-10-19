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
			name:        "should return internal cluster client",
			clusterName: "external-cluster",
			internalClusterClient: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, cdPipeApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "secret",
								Namespace: "default",
								Labels:    map[string]string{argocdClusterSecretLabel: argocdClusterSecretLabelVal},
							},
							Data: map[string][]byte{
								"config": []byte(`{"bearerToken": "token"}`),
								"name":   []byte("external-cluster"),
								"server": []byte("https://external-cluster"),
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
				require.NoError(t, cdPipeApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						&corev1.Secret{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "secret",
								Namespace: "default",
								Labels:    map[string]string{argocdClusterSecretLabel: argocdClusterSecretLabelVal},
							},
							Data: map[string][]byte{
								"config": []byte(`not json data`),
								"name":   []byte("external-cluster"),
							},
						},
					).
					Build()
			},
			want: require.Nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to unmarshal cluster config")
			},
		},
		{
			name:        "secret not found",
			clusterName: "external-cluster",
			internalClusterClient: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, cdPipeApi.AddToScheme(s))
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
				require.NoError(t, cdPipeApi.AddToScheme(s))
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
