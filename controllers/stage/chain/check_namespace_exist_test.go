package chain

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

func TestCheckNamespaceExist_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name    string
		stage   *cdPipeApi.Stage
		objects []client.Object
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "namespace exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
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
		},
		{
			name: "namespace doesn't exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "doesn't exist")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := CheckNamespaceExist{
				client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build(),
				log:    logr.Discard(),
			}

			tt.wantErr(t, h.ServeRequest(tt.stage))
		})
	}
}
