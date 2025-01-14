package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestStageValidationWebhook_ValidateCreate(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, pipelineApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name    string
		obj     runtime.Object
		client  func(t *testing.T) client.Client
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "valid stage, no namespace conflict",
			obj: &pipelineApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage1",
					Namespace: "default",
				},
				Spec: pipelineApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipeline",
					ClusterName: pipelineApi.InCluster,
					Namespace:   "stage1-ns",
				},
			},
			client: func(t *testing.T) client.Client {
				stageWithDifferentTargetNs := &pipelineApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "stage2",
						Namespace: "default",
					},
					Spec: pipelineApi.StageSpec{
						Name:        "stage2",
						CdPipeline:  "pipeline",
						ClusterName: pipelineApi.InCluster,
						Namespace:   "stage2-ns",
					},
				}

				stageWithDifferentClusterButSameTargetNs := &pipelineApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "stage3",
						Namespace: "default",
					},
					Spec: pipelineApi.StageSpec{
						Name:        "stage3",
						CdPipeline:  "pipeline",
						ClusterName: "cluster2",
						Namespace:   "stage1-ns",
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					stageWithDifferentTargetNs,
					stageWithDifferentClusterButSameTargetNs,
				).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "namespace conflict",
			obj: &pipelineApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage1",
					Namespace: "default",
				},
				Spec: pipelineApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipeline",
					ClusterName: pipelineApi.InCluster,
					Namespace:   "stage1-ns",
				},
			},
			client: func(t *testing.T) client.Client {
				stageWithSameTargetNs := &pipelineApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "stage2",
						Namespace: "default",
					},
					Spec: pipelineApi.StageSpec{
						Name:        "stage2",
						CdPipeline:  "pipeline",
						ClusterName: pipelineApi.InCluster,
						Namespace:   "stage1-ns",
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					stageWithSameTargetNs,
				).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "namespace stage1-ns is already used in CDPipeline pipeline Stage stage2")
			},
		},
		{
			name: "namespace already used in the cluster",
			obj: &pipelineApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage1",
					Namespace: "default",
				},
				Spec: pipelineApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipeline",
					ClusterName: pipelineApi.InCluster,
					Namespace:   "ns1",
				},
			},
			client: func(t *testing.T) client.Client {
				ns1 := &corev1.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: "ns1",
						Labels: map[string]string{
							util.TenantLabelName: "ns1",
						},
					},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns1).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "namespace ns1 is already used in the cluster")
			},
		},
		{
			name: "invalid object given",
			obj:  &codebaseApi.Codebase{},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "the wrong object given, expected Stage")
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewStageValidationWebhook(tt.client(t))

			err := r.ValidateCreate(context.Background(), tt.obj)
			tt.wantErr(t, err)
		})
	}
}

func TestStageValidationWebhook_ValidateUpdate(t *testing.T) {
	r := NewStageValidationWebhook(fake.NewClientBuilder().Build())

	assert.NoError(t, r.ValidateUpdate(context.Background(), &pipelineApi.Stage{}, &pipelineApi.Stage{}))
}

func TestStageValidationWebhook_ValidateDelete(t *testing.T) {
	r := NewStageValidationWebhook(fake.NewClientBuilder().Build())

	assert.NoError(t, r.ValidateDelete(context.Background(), &pipelineApi.Stage{}))
}
