package argocd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/aws"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/aws/mocks"
)

func TestAddClusterLabel(t *testing.T) {
	s := &corev1.Secret{}

	AddClusterLabel(s)

	assert.Equal(t, ClusterLabelVal, s.GetLabels()[ClusterLabel])
}

func TestArgoIRSAClusterSecretToKubeconfig(t *testing.T) {
	t.Parallel()

	exp := time.Now().Add(time.Minute * 15)

	tests := []struct {
		name           string
		secret         *corev1.Secret
		tokenGenerator func(t *testing.T) aws.AIMAuthTokenGenerator
		want           func(t *testing.T, conf []byte)
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "successful conversion",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"config": []byte(`{"awsAuthConfig":{"clusterName":"test","roleARN":"arn:aws:iam::123456789012:role/test"},"tlsClientConfig":{"insecure":true}}`),
					"server": []byte("https://test-cluster"),
				},
			},
			tokenGenerator: func(t *testing.T) aws.AIMAuthTokenGenerator {
				m := mocks.NewMockAIMAuthTokenGenerator(t)

				m.On("GetWithRole", "test", "arn:aws:iam::123456789012:role/test").
					Return(aws.Token{
						Token:      "token",
						Expiration: exp,
					}, nil)

				return m
			},
			want: func(t *testing.T, conf []byte) {
				require.NotEmpty(t, conf)

				config, err := clientcmd.RESTConfigFromKubeConfig(conf)
				require.NoError(t, err)

				assert.Equal(t, "https://test-cluster", config.Host)
				assert.Equal(t, "token", config.BearerToken)

			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid secret config",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"config": []byte(`not json`),
					"server": []byte("https://test-cluster"),
				},
			},
			tokenGenerator: func(t *testing.T) aws.AIMAuthTokenGenerator {
				return mocks.NewMockAIMAuthTokenGenerator(t)
			},
			want:    func(t *testing.T, conf []byte) {},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ArgoIRSAClusterSecretToKubeconfig(tt.secret, tt.tokenGenerator(t))
			tt.wantErr(t, err)
			tt.want(t, got)
		})
	}
}
