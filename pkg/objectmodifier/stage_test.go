package objectmodifier

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

func Test_setStageLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stage     *cdPipeApi.Stage
		wantStage *cdPipeApi.Stage
		want      bool
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "should return true if stage label was set",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
				},
			},
			wantStage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
					Labels: map[string]string{
						cdPipeApi.CodebaseTypeLabelName: "test-pipeline",
					},
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
				},
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name: "should return false if stage label already exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
					Labels: map[string]string{
						cdPipeApi.CodebaseTypeLabelName: "test-pipeline",
					},
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
				},
			},
			wantStage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
					Labels: map[string]string{
						cdPipeApi.CodebaseTypeLabelName: "test-pipeline",
					},
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
				},
			},
			want:    false,
			wantErr: require.NoError,
		},
		{
			name:      "stage is nil",
			stage:     nil,
			wantStage: nil,
			want:      false,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "stage is nil")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := setStageLabel(logr.NewContext(context.Background(), logr.Discard()), tt.stage)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantStage, tt.stage)
			tt.wantErr(t, err)
		})
	}
}

func Test_updateStageNamespaceSpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stage     *cdPipeApi.Stage
		wantStage *cdPipeApi.Stage
		want      bool
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "should return true if stage namespace was updated",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
				},
			},
			wantStage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
					Namespace: util.GenerateNamespaceName(&cdPipeApi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-stage",
							Namespace: "default",
						},
					}),
				},
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name: "should return false if stage namespace already exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
					Namespace: util.GenerateNamespaceName(&cdPipeApi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-stage",
							Namespace: "default",
						},
					}),
				},
			},
			wantStage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
					Namespace: util.GenerateNamespaceName(&cdPipeApi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-stage",
							Namespace: "default",
						},
					}),
				},
			},
			want:    false,
			wantErr: require.NoError,
		},
		{
			name:      "stage is nil",
			stage:     nil,
			wantStage: nil,
			want:      false,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "stage is nil")
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := updateStageNamespaceSpec(logr.NewContext(context.Background(), logr.Discard()), tt.stage)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantStage, tt.stage)
			tt.wantErr(t, err)
		})
	}
}

func Test_stageOwnerRefModifier_Apply(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, cdPipeApi.AddToScheme(scheme))

	tests := []struct {
		name      string
		stage     *cdPipeApi.Stage
		objects   []runtime.Object
		want      bool
		wantErr   require.ErrorAssertionFunc
		wantCheck func(t *testing.T, stage *cdPipeApi.Stage)
	}{
		{
			name: "should return true if stage owner reference was added",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
				},
			},
			objects: []runtime.Object{
				&cdPipeApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pipeline",
						Namespace: "default",
					},
				},
			},
			want:    true,
			wantErr: require.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage) {
				require.Len(t, stage.OwnerReferences, 1)
				assert.Equal(t, "test-pipeline", stage.OwnerReferences[0].Name)
			},
		},
		{
			name: "should return false if stage owner reference already exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "test-pipeline",
							UID:        types.UID("64fbdcc6-c176-41a9-8d8c-f5c0a955acd8"),
							Controller: pointer.Bool(true),
							Kind:       consts.CDPipelineKind,
						},
					},
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
				},
			},
			objects: []runtime.Object{
				&cdPipeApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pipeline",
						Namespace: "default",
						UID:       types.UID("64fbdcc6-c176-41a9-8d8c-f5c0a955acd8"),
					},
				},
			},
			want:    false,
			wantErr: require.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage) {
				require.Len(t, stage.OwnerReferences, 1)
				assert.Equal(t, "test-pipeline", stage.OwnerReferences[0].Name)
			},
		},
		{
			name: "cd pipeline not found",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
				},
			},
			want: false,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cdpipeline test-pipeline doesn't exist")
			},
		},
		{
			name:  "stage is nil",
			stage: nil,
			want:  false,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "stage is nil")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newStageOwnerRefModifier(
				fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				scheme,
			)
			got, err := m.Apply(logr.NewContext(context.Background(), logr.Discard()), tt.stage)
			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}

func TestStageBatchModifier_Apply(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, cdPipeApi.AddToScheme(scheme))

	tests := []struct {
		name    string
		stage   *cdPipeApi.Stage
		objects []runtime.Object
		want    bool
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should return true if one of the modifiers returns true",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
					Namespace: util.GenerateNamespaceName(&cdPipeApi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-stage",
							Namespace: "default",
						},
					}),
				},
			},
			objects: []runtime.Object{
				&cdPipeApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pipeline",
						Namespace: "default",
					},
				},
				&cdPipeApi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-stage",
						Namespace: "default",
					},
				},
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name: "should return false if all modifiers return false",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
					Labels: map[string]string{
						cdPipeApi.CodebaseTypeLabelName: "test-pipeline",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "test-pipeline",
							UID:        types.UID("64fbdcc6-c176-41a9-8d8c-f5c0a955acd8"),
							Controller: pointer.Bool(true),
							Kind:       consts.CDPipelineKind,
						},
					},
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
					Namespace: util.GenerateNamespaceName(&cdPipeApi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-stage",
							Namespace: "default",
						},
					}),
				},
			},
			objects: []runtime.Object{
				&cdPipeApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pipeline",
						Namespace: "default",
					},
				},
				&cdPipeApi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-stage",
						Namespace: "default",
					},
				},
			},
			want:    false,
			wantErr: require.NoError,
		},
		{
			name: "should return error if one of the modifiers returns error",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
				},
			},
			want: false,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "cdpipeline test-pipeline doesn't exist")
			},
		},
		{
			name: "failed to patch stage",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "test-pipeline",
							UID:        types.UID("64fbdcc6-c176-41a9-8d8c-f5c0a955acd8"),
							Controller: pointer.Bool(true),
							Kind:       consts.CDPipelineKind,
						},
					},
				},
				Spec: cdPipeApi.StageSpec{
					CdPipeline: "test-pipeline",
					Namespace: util.GenerateNamespaceName(&cdPipeApi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-stage",
							Namespace: "default",
						},
					}),
				},
			},
			objects: []runtime.Object{
				&cdPipeApi.CDPipeline{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pipeline",
						Namespace: "default",
					},
				},
			},
			want: false,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to patch stage")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewStageBatchModifierAll(
				fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build(),
				scheme,
			)
			got, err := m.Apply(logr.NewContext(context.Background(), logr.Discard()), tt.stage)
			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}

func TestNewStageBatchModifier(t *testing.T) {
	got := NewStageBatchModifier(fake.NewClientBuilder().Build(), []StageModifier{})
	assert.NotNil(t, got)
}
