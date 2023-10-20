package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

func TestDelegateNamespaceDeletion_ServeRequest(t *testing.T) {
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
			name: "deletion of project is successful",
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
			objects: []client.Object{
				&projectApi.Project{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "default-stage-1",
					},
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				err := c.Get(
					context.Background(),
					client.ObjectKey{Name: s.Spec.Namespace}, &projectApi.Project{},
				)
				require.Error(t, err)
				require.True(t, apiErrors.IsNotFound(err))
			},
		},
		{
			name: "deletion of namespace is successful",
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
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "default-stage-1",
					},
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				err := c.Get(
					context.Background(),
					client.ObjectKey{Name: s.Spec.Namespace}, &corev1.Namespace{},
				)
				require.Error(t, err)
				require.True(t, apiErrors.IsNotFound(err))
			},
		},
		{
			name: "deletion of kiosk space is successful",
			prepare: func(t *testing.T) {
				t.Setenv(platform.TenancyEngineEnv, platform.TenancyEngineKiosk)
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
			objects: []client.Object{
				kiosk.NewKioskSpace(map[string]interface{}{
					"name": "default-stage-1",
				}),
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				err := c.Get(
					context.Background(),
					client.ObjectKey{Name: s.Spec.Namespace}, &corev1.Namespace{},
				)
				require.Error(t, err)
				require.True(t, apiErrors.IsNotFound(err))
			},
		},
		{
			name:    "no platform env is set, default is kubernetes",
			prepare: func(t *testing.T) {},
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
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "default-stage-1",
					},
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				err := c.Get(
					context.Background(),
					client.ObjectKey{Name: s.Spec.Namespace}, &corev1.Namespace{},
				)
				require.Error(t, err)
			},
		},
		{
			name: "external cluster is set",
			prepare: func(t *testing.T) {
				t.Setenv(platform.TenancyEngineEnv, platform.TenancyEngineKiosk)
				t.Setenv(platform.TypeEnv, platform.Kubernetes)
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
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "default-stage-1",
					},
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				err := c.Get(
					context.Background(),
					client.ObjectKey{Name: s.Spec.Namespace}, &corev1.Namespace{},
				)
				require.Error(t, err)
			},
		},
		{
			name: "namespace is not managed by operator",
			prepare: func(t *testing.T) {
				t.Setenv(platform.TypeEnv, platform.Kubernetes)
				t.Setenv(platform.ManageNamespaceEnv, "false")
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
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "default-stage-1",
					},
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				err := c.Get(
					context.Background(),
					client.ObjectKey{Name: s.Spec.Namespace}, &corev1.Namespace{},
				)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			c := DelegateNamespaceDeletion{
				multiClusterClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build(),
				log:                logr.Discard(),
			}

			err := c.ServeRequest(tt.stage)
			tt.wantErr(t, err)
			tt.wantAssert(t, c.multiClusterClient, tt.stage)
		})
	}
}
