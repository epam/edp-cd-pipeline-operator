package webhook

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

func TestCDPipelineValidationWebhook_ValidateUpdate(t *testing.T) {
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
			name: "validating CDPipeline update with protected label",
			args: args{
				oldObj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
						Labels: map[string]string{
							protectedLabel: fmt.Sprintf("%s-%s", updateOperation, deleteOperation),
						},
					},
					Spec: pipelineApi.CDPipelineSpec{
						Description: "cd-pipeline",
					},
				},
				newObj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
						Labels: map[string]string{
							protectedLabel: fmt.Sprintf("%s-%s", updateOperation, deleteOperation),
						},
					},
					Spec: pipelineApi.CDPipelineSpec{
						Description: "cd-pipeline 2",
					},
				},
			},
			wantErr: require.Error,
		},
		{
			name: "validating CDPipeline update without protected label in new object",
			args: args{
				oldObj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
						Labels: map[string]string{
							protectedLabel: fmt.Sprintf("%s-%s", updateOperation, deleteOperation),
						},
					},
					Spec: pipelineApi.CDPipelineSpec{
						Description: "cd-pipeline",
					},
				},
				newObj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
					},
					Spec: pipelineApi.CDPipelineSpec{
						Description: "cd-pipeline 2",
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name: "validating CDPipeline update without protected label in old object",
			args: args{
				oldObj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
					},
					Spec: pipelineApi.CDPipelineSpec{
						Description: "cd-pipeline",
					},
				},
				newObj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
						Labels: map[string]string{
							protectedLabel: fmt.Sprintf("%s-%s", updateOperation, deleteOperation),
						},
					},
					Spec: pipelineApi.CDPipelineSpec{
						Description: "cd-pipeline 2",
					},
				},
			},
			wantErr: require.Error,
		},
		{
			name: "validating CDPipeline update with protected label(delete)",
			args: args{
				oldObj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
						Labels: map[string]string{
							protectedLabel: deleteOperation,
						},
					},
					Spec: pipelineApi.CDPipelineSpec{
						Description: "cd-pipeline",
					},
				},
				newObj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
						Labels: map[string]string{
							protectedLabel: deleteOperation,
						},
					},
					Spec: pipelineApi.CDPipelineSpec{
						Description: "cd-pipeline 2",
					},
				},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := NewCDPipelineValidationWebhook(cl)
			w, err := cd.ValidateUpdate(context.Background(), tt.args.oldObj, tt.args.newObj)
			assert.Nil(t, w)
			tt.wantErr(t, err)
		})
	}
}

func TestCDPipelineValidationWebhook_ValidateDelete(t *testing.T) {
	cl := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()

	type args struct {
		obj runtime.Object
	}

	tests := []struct {
		name    string
		args    args
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "validating CDPipeline delete with protected label",
			args: args{
				obj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
						Labels: map[string]string{
							protectedLabel: deleteOperation,
						},
					},
				},
			},
			wantErr: require.Error,
		},
		{
			name: "validating CDPipeline delete without protected label",
			args: args{
				obj: &pipelineApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cd-pipeline",
						Labels: map[string]string{
							protectedLabel: updateOperation,
						},
					},
				},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := NewCDPipelineValidationWebhook(cl)
			w, err := cd.ValidateDelete(context.Background(), tt.args.obj)
			assert.Nil(t, w)
			tt.wantErr(t, err)
		})
	}
}
