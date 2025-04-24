package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

func TestPutOpenshiftProject_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	require.NoError(t, projectApi.AddToScheme(scheme))

	tests := []struct {
		name       string
		stage      *cdPipeApi.Stage
		objects    []client.Object
		wantErr    require.ErrorAssertionFunc
		wantAssert func(t *testing.T, c client.Client, s *cdPipeApi.Stage)
	}{
		{
			name: "creation of project is successful",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "stage-1-ns",
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(
					t,
					c.Get(
						context.Background(),
						types.NamespacedName{Name: s.Spec.Namespace}, &projectApi.ProjectRequest{},
					),
				)
			},
		},
		{
			name: "project already exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					Namespace: "stage-1-ns",
				},
			},
			objects: []client.Object{
				&projectApi.ProjectRequest{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "stage-1-ns",
					},
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(
					t,
					c.Get(
						context.Background(),
						types.NamespacedName{Name: s.Spec.Namespace}, &projectApi.ProjectRequest{},
					),
				)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()

			c := PutOpenshiftProject{
				client: k8sClient,
			}

			err := c.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage)
			tt.wantErr(t, err)
			tt.wantAssert(t, k8sClient, tt.stage)
		})
	}
}
