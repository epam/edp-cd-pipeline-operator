package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

func TestPutConfigMap_ServeRequest(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, cdPipeApi.AddToScheme(scheme))

	tests := []struct {
		name    string
		stage   *cdPipeApi.Stage
		client  func(t *testing.T) client.Client
		wantErr require.ErrorAssertionFunc
		want    func(t *testing.T, stage *cdPipeApi.Stage, cl client.Client)
	}{
		{
			name: "create configmap successfully",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe-dev",
					Namespace: "default",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: require.NoError,
			want: func(t *testing.T, stage *cdPipeApi.Stage, cl client.Client) {
				cm := &corev1.ConfigMap{}
				require.NoError(t, cl.Get(
					context.Background(),
					client.ObjectKey{Namespace: stage.Namespace, Name: stage.Name},
					cm,
				))
			},
		},
		{
			name: "configmap already exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe-dev",
					Namespace: "default",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe-dev",
								Namespace: "default",
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := tt.client(t)
			h := NewPutConfigMap(cl)

			tt.wantErr(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage))

			if tt.want != nil {
				tt.want(t, tt.stage, cl)
			}
		})
	}
}
