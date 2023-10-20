package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project/v1"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

func TestDeleteOpenshiftProject_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	require.NoError(t, projectApi.AddToScheme(scheme))

	tests := []struct {
		name       string
		stage      *cdPipeApi.Stage
		objects    []runtime.Object
		wantErr    require.ErrorAssertionFunc
		wantAssert func(t *testing.T, c client.Client, s *cdPipeApi.Stage)
	}{
		{
			name: "deletion of project is successful",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
			},
			objects: []runtime.Object{
				&projectApi.Project{
					ObjectMeta: metaV1.ObjectMeta{
						Name: util.GenerateNamespaceName(&cdPipeApi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Name:      "stage-1",
								Namespace: "default",
							},
						}),
					},
				},
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {
				require.NoError(
					t,
					client.IgnoreNotFound(
						c.Get(context.Background(), client.ObjectKey{Name: util.GenerateNamespaceName(s)}, &projectApi.Project{}),
					),
				)
			},
		},
		{
			name: "project is not found, skip deletion",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
			},
			wantErr:    require.NoError,
			wantAssert: func(t *testing.T, c client.Client, s *cdPipeApi.Stage) {},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := DeleteOpenshiftProject{
				multiClusterClient: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				log:                logr.Discard(),
			}

			tt.wantErr(t, h.ServeRequest(tt.stage))
			tt.wantAssert(t, h.multiClusterClient, tt.stage)
		})
	}
}
