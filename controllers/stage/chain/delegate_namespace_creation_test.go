package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

func TestDelegateNamespaceCreation_ServeRequest(t *testing.T) {
	scheme := runtime.NewScheme()

	require.NoError(t, projectApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name        string
		stage       *cdPipeApi.Stage
		platformEnv string
		wantErr     require.ErrorAssertionFunc
		wantAssert  func(t *testing.T, c client.Client, s *cdPipeApi.Stage)
	}{
		{
			name:        "creation of project is successful",
			platformEnv: "openshift",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(t,
					c.Get(
						context.Background(),
						client.ObjectKey{Name: util.GenerateNamespaceName(s)}, &projectApi.Project{},
					),
				)
			},
		},
		{
			name:        "creation of namespace is successful",
			platformEnv: "kubernetes",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(t,
					c.Get(
						context.Background(),
						client.ObjectKey{Name: util.GenerateNamespaceName(s)}, &corev1.Namespace{},
					),
				)
			},
		},
		{
			name: "no platform env is set, default is openshift",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(t,
					c.Get(
						context.Background(),
						client.ObjectKey{Name: util.GenerateNamespaceName(s)}, &projectApi.Project{},
					),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PLATFORM_TYPE", tt.platformEnv)

			c := DelegateNamespaceCreation{
				client: fake.NewClientBuilder().WithScheme(scheme).Build(),
				log:    logr.Discard(),
			}

			err := c.ServeRequest(tt.stage)
			tt.wantErr(t, err)
			tt.wantAssert(t, c.client, tt.stage)
		})
	}
}
