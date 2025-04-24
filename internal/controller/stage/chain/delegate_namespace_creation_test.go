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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

func TestDelegateNamespaceCreation_ServeRequest(t *testing.T) {
	scheme := runtime.NewScheme()

	require.NoError(t, projectApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name       string
		stage      *cdPipeApi.Stage
		prepare    func(t *testing.T)
		objects    []client.Object
		wantErr    require.ErrorAssertionFunc
		wantAssert func(t *testing.T, c client.Client, s *cdPipeApi.Stage)
	}{
		{
			name: "creation of project is successful",
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Openshift)
			},
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace:   "default-stage-1",
					ClusterName: cdPipeApi.InCluster,
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(t,
					c.Get(
						context.Background(),
						client.ObjectKey{Name: s.Spec.Namespace}, &projectApi.ProjectRequest{},
					),
				)
			},
		},
		{
			name: "creation of namespace is successful",
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Kubernetes)
			},
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace:   "default-stage-1",
					ClusterName: cdPipeApi.InCluster,
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(t,
					c.Get(
						context.Background(),
						client.ObjectKey{Name: s.Spec.Namespace}, &corev1.Namespace{},
					),
				)
			},
		},
		{
			name: "no platform env is set, default is kubernetes",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace:   "default-stage-1",
					ClusterName: cdPipeApi.InCluster,
				},
			},
			prepare: func(t *testing.T) {
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(t,
					c.Get(
						context.Background(),
						client.ObjectKey{Name: s.Spec.Namespace}, &corev1.Namespace{},
					),
				)
			},
		},
		{
			name: "namespace is not managed by the operator",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace:   "default-stage-1",
					ClusterName: cdPipeApi.InCluster,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.ManageNamespaceEnv, "false")
				t.Setenv(platform.TypeEnv, platform.Kubernetes)
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "default-stage-1",
					},
				},
			},
			wantErr:    require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {},
		},
		{
			name: "namespace is not managed by the operator and doesn't exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace:   "default-stage-1",
					ClusterName: cdPipeApi.InCluster,
				},
			},
			prepare: func(t *testing.T) {
				t.Setenv(platform.ManageNamespaceEnv, "false")
				t.Setenv(platform.TypeEnv, platform.Kubernetes)
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "doesn't exist")
			},
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {},
		},
		{
			name: "skip multi-tenancy engines for kubernetes platform",
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Kubernetes)
				t.Setenv(platform.ManageNamespaceEnv, "true")
				t.Setenv(platform.TenancyEngineEnv, platform.TenancyEngineCapsule)
			},
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace:   "default-stage-1",
					ClusterName: "external-cluster",
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(t,
					c.Get(
						context.Background(),
						client.ObjectKey{Name: s.Spec.Namespace}, &corev1.Namespace{},
					),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			c := DelegateNamespaceCreation{
				client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build(),
			}

			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage)
			tt.wantErr(t, err)
			tt.wantAssert(t, c.client, tt.stage)
		})
	}
}
