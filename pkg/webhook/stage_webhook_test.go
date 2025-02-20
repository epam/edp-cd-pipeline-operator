package webhook

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
				ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
				ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
	cl := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()

	type args struct {
		oldObj runtime.Object
		newObj runtime.Object
	}

	tests := []struct {
		name    string
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "validating Stage update with protected label",
			args: args{
				oldObj: &pipelineApi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name: "stage",
						Labels: map[string]string{
							protectedLabel: fmt.Sprintf("%s-%s", updateOperation, deleteOperation),
						},
					},
					Spec: pipelineApi.StageSpec{
						Description: "stage",
					},
				},
				newObj: &pipelineApi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name: "stage",
						Labels: map[string]string{
							protectedLabel: fmt.Sprintf("%s-%s", updateOperation, deleteOperation),
						},
					},
					Spec: pipelineApi.StageSpec{
						Description: "stage 2",
					},
				},
			},
			wantErr: require.Error,
		},
		{
			name: "validating Stage update without protected label",
			args: args{
				oldObj: &pipelineApi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name: "stage",
					},
					Spec: pipelineApi.StageSpec{
						Description: "stage",
					},
				},
				newObj: &pipelineApi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name: "stage",
					},
					Spec: pipelineApi.StageSpec{
						Description: "stage 2",
					},
				},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := NewStageValidationWebhook(cl)
			tt.wantErr(t, cd.ValidateUpdate(context.Background(), tt.args.oldObj, tt.args.newObj))
		})
	}
}

func TestStageValidationWebhook_ValidateDelete(t *testing.T) {
	cl := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()

	type args struct {
		obj runtime.Object
	}

	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "validating Stage delete with protected label",
			args: args{
				obj: &pipelineApi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name: "stage",
						Labels: map[string]string{
							protectedLabel: deleteOperation,
						},
					},
				},
			},
			wantErr: assert.Error,
		},
		{
			name: "validating Stage delete without protected label",
			args: args{
				obj: &pipelineApi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name: "stage",
					},
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := NewStageValidationWebhook(cl)
			tt.wantErr(t, st.ValidateDelete(context.Background(), tt.args.obj))
		})
	}
}
