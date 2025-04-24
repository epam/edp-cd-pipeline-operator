package stage

import (
	"context"
	"fmt"
	"testing"
	"time"

	argoApi "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	projectApi "github.com/openshift/api/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/objectmodifier"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

const (
	cdPipeline      = "stub-cdPipeline-name"
	dockerImageName = "docker-image-name"
	name            = "stub-name"
	namespace       = "stub-namespace"
	labelValue      = "stub-data"
)

func getStage(t *testing.T, c client.Client, name string) *cdPipeApi.Stage {
	t.Helper()

	stage := &cdPipeApi.Stage{}
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, stage); err != nil {
		t.Fatal(err)
	}

	return stage
}

func createLabelName() string {
	return fmt.Sprintf("%s/%s", name, name)
}

func TestTryToDeleteCDStage_DeletionTimestampIsZero(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.Stage{})

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Finalizers: []string{},
		},
		Spec: cdPipeApi.StageSpec{
			TriggerType: cdPipeApi.TriggerTypeAutoDeploy,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stage).Build()

	reconcileStage := ReconcileStage{
		client: fakeClient,
		scheme: scheme,
		log:    logr.Discard(),
	}

	_, err := reconcileStage.tryToDeleteCDStage(ctrl.LoggerInto(context.Background(), logr.Discard()), stage)
	assert.NoError(t, err)

	stageAfterReconcile := getStage(t, reconcileStage.client, name)
	assert.Equal(t, stageAfterReconcile.Finalizers, []string{envLabelDeletionFinalizer})
}

func TestTryToDeleteCDStage_Success(t *testing.T) {
	scheme := runtime.NewScheme()

	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, projectApi.Install(scheme))
	require.NoError(t, argoApi.AddToScheme(scheme))

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Generation: 0,
			DeletionTimestamp: &metaV1.Time{
				Time: time.Now().UTC(),
			},
			Finalizers: []string{envLabelDeletionFinalizer},
		},
		Spec: cdPipeApi.StageSpec{
			Name:        name,
			CdPipeline:  cdPipeline,
			TriggerType: cdPipeApi.TriggerTypeAutoDeploy,
			Order:       0,
		},
	}

	labels := make(map[string]string)
	labels[createLabelName()] = labelValue
	labels[cluster.CodebaseBranchLabel] = dockerImageName

	cdPipeline := &cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{dockerImageName},
			Name:               name,
		},
	}

	image := &codebaseApi.CodebaseImageStream{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline, image, stage).Build()

	reconcileStage := ReconcileStage{
		client: fakeClient,
		scheme: scheme,
		log:    logr.Discard(),
	}

	_, err := reconcileStage.tryToDeleteCDStage(ctrl.LoggerInto(context.Background(), logr.Discard()), stage)
	assert.NoError(t, err)

	previousImageStream := &codebaseApi.CodebaseImageStream{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      dockerImageName,
	}, previousImageStream)
	require.NoError(t, err)
	assert.Empty(t, previousImageStream.Labels[createLabelName()])

	_ = &cdPipeApi.Stage{}
	err = fakeClient.Get(
		context.Background(),
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		stage,
	)
	require.Error(t, err)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestTryToDeleteCDStage_PostponeDeletion(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, cdPipeApi.AddToScheme(scheme))

	stageToRemove := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:              "stage-name",
			Namespace:         "stage-namespace",
			Labels:            map[string]string{cdPipeApi.StageCdPipelineLabelName: "cd-pipeline-name"},
			Finalizers:        []string{envLabelDeletionFinalizer},
			DeletionTimestamp: &metaV1.Time{Time: time.Now().UTC()},
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: "cd-pipeline-name",
			Order:      0,
		},
	}
	parentStage := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage-name2",
			Namespace: "stage-namespace",
			Labels:    map[string]string{cdPipeApi.StageCdPipelineLabelName: "cd-pipeline-name"},
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: "cd-pipeline-name",
			Order:      1,
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stageToRemove, parentStage).Build()

	controller := NewReconcileStage(
		k8sClient,
		scheme,
		logr.Discard(),
		objectmodifier.NewStageBatchModifier(k8sClient, []objectmodifier.StageModifier{}),
	)

	res, err := controller.tryToDeleteCDStage(ctrl.LoggerInto(context.Background(), logr.Discard()), stageToRemove)
	require.NoError(t, err)
	assert.Equal(t, &reconcile.Result{RequeueAfter: waitForParentStagesDeletion}, res)
}

func TestSetFinishStatus_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.Stage{})

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stage).Build()

	reconcileStage := ReconcileStage{
		client: fakeClient,
		scheme: scheme,
		log:    logr.Discard(),
	}

	err := reconcileStage.setFinishStatus(context.Background(), stage)
	assert.NoError(t, err)

	stageAfterReconcile := getStage(t, reconcileStage.client, name)
	assert.Equal(t, consts.FinishedStatus, stageAfterReconcile.Status.Status)
}

func TestReconcileStage_Reconcile_Success(t *testing.T) {
	scheme := runtime.NewScheme()

	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, projectApi.Install(scheme))
	require.NoError(t, argoApi.AddToScheme(scheme))

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Generation: 0,
			DeletionTimestamp: &metaV1.Time{
				Time: time.Now().UTC(),
			},
			Finalizers: []string{envLabelDeletionFinalizer},
		},
		Spec: cdPipeApi.StageSpec{
			Name:        name,
			CdPipeline:  cdPipeline,
			TriggerType: cdPipeApi.TriggerTypeAutoDeploy,
			Order:       0,
		},
	}

	labels := make(map[string]string)
	labels[createLabelName()] = labelValue
	labels[cluster.CodebaseBranchLabel] = dockerImageName

	cdPipeline := &cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{dockerImageName},
			Name:               name,
		},
	}

	image := &codebaseApi.CodebaseImageStream{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline, image, stage).Build()

	reconcileStage := NewReconcileStage(
		fakeClient,
		scheme,
		logr.Discard(),
		objectmodifier.NewStageBatchModifier(fakeClient, []objectmodifier.StageModifier{}),
	)

	_, err := reconcileStage.Reconcile(ctrl.LoggerInto(context.Background(), logr.Discard()), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}})
	require.NoError(t, err)

	previousImageStream := &codebaseApi.CodebaseImageStream{}
	err = reconcileStage.client.Get(context.Background(),
		types.NamespacedName{
			Namespace: namespace,
			Name:      dockerImageName,
		},
		previousImageStream,
	)
	require.NoError(t, err)
	assert.Empty(t, previousImageStream.Labels[createLabelName()])

	_ = &cdPipeApi.Stage{}
	err = fakeClient.Get(
		context.Background(),
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		stage,
	)
	require.Error(t, err)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestReconcileStage_ReconcileReconcile_SetOwnerRef(t *testing.T) {
	scheme := runtime.NewScheme()

	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))
	require.NoError(t, k8sApi.AddToScheme(scheme))
	require.NoError(t, argoApi.AddToScheme(scheme))

	cm := &corev1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      platform.KrciConfigMap,
			Namespace: namespace,
		},
		Data: map[string]string{
			platform.KrciConfigContainerRegistryHost:  "test-registry",
			platform.KrciConfigContainerRegistrySpace: "test-space",
		},
	}

	qualityGate := cdPipeApi.QualityGate{}

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Finalizers: []string{envLabelDeletionFinalizer},
		},
		Spec: cdPipeApi.StageSpec{
			Name:         name,
			CdPipeline:   cdPipeline,
			TriggerType:  cdPipeApi.TriggerTypeAutoDeploy,
			Order:        0,
			QualityGates: []cdPipeApi.QualityGate{qualityGate},
			Namespace:    "stub-namespace",
		},
	}

	labels := make(map[string]string)
	labels[createLabelName()] = labelValue
	labels[cluster.CodebaseBranchLabel] = dockerImageName

	cdPipeline := &cdPipeApi.CDPipeline{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{dockerImageName},
			Name:               name,
		},
	}

	image := &codebaseApi.CodebaseImageStream{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerImageName,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	appset := &argoApi.ApplicationSet{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cdPipeline.Name,
			Namespace: namespace,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cdPipeline, image, stage, cm, appset).Build()

	reconcileStage := NewReconcileStage(
		fakeClient,
		scheme,
		logr.Discard(),
		objectmodifier.NewStageBatchModifierAll(fakeClient, scheme),
	)

	_, err := reconcileStage.Reconcile(ctrl.LoggerInto(context.Background(), logr.Discard()), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}})
	require.NoError(t, err)

	expectedLabels := map[string]string{
		"app.edp.epam.com/cdPipelineName": cdPipeline.Name,
	}

	stageAfterReconcile := getStage(t, reconcileStage.client, name)
	assert.Equal(t, cdPipeline.Name, stageAfterReconcile.OwnerReferences[0].Name)
	assert.Equal(t, expectedLabels, stageAfterReconcile.Labels)
}

func TestReconcileStage_Reconcile_StageIsNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.Stage{})

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconcileStage := ReconcileStage{
		client: fakeClient,
		scheme: scheme,
		log:    logr.Discard(),
	}

	stage := &cdPipeApi.Stage{}
	err := reconcileStage.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, stage)
	assert.True(t, k8sErrors.IsNotFound(err))

	_, err = reconcileStage.Reconcile(ctrl.LoggerInto(context.Background(), logr.Discard()), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}})
	assert.NoError(t, err)
}

func TestReconcileStage_isLastStage(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, cdPipeApi.AddToScheme(scheme))

	tests := []struct {
		name    string
		stage   *cdPipeApi.Stage
		objects []client.Object
		want    bool
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "should return true when stage is last",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "ns-1",
				},
				Spec: cdPipeApi.StageSpec{
					Order:      1,
					CdPipeline: "cd-pipeline-1",
				},
			},
			objects: []client.Object{
				&cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "stage-2",
						Namespace: "ns-1",
						Labels:    map[string]string{cdPipeApi.StageCdPipelineLabelName: "cd-pipeline-1"},
					},
					Spec: cdPipeApi.StageSpec{
						Order: 0,
					},
				},
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name: "should return true when order is equal",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "ns-1",
				},
				Spec: cdPipeApi.StageSpec{
					Order:      1,
					CdPipeline: "cd-pipeline-1",
				},
			},
			objects: []client.Object{
				&cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "stage-2",
						Namespace: "ns-1",
						Labels:    map[string]string{cdPipeApi.StageCdPipelineLabelName: "cd-pipeline-1"},
					},
					Spec: cdPipeApi.StageSpec{
						Order: 1,
					},
				},
			},
			want:    true,
			wantErr: require.NoError,
		},
		{
			name: "should return false when stage is not last",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "ns-1",
				},
				Spec: cdPipeApi.StageSpec{
					Order:      0,
					CdPipeline: "cd-pipeline-1",
				},
			},
			objects: []client.Object{
				&cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      "stage-2",
						Namespace: "ns-1",
						Labels:    map[string]string{cdPipeApi.StageCdPipelineLabelName: "cd-pipeline-1"},
					},
					Spec: cdPipeApi.StageSpec{
						Order: 1,
					},
				},
			},
			want:    false,
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
			r := NewReconcileStage(
				k8sClient,
				scheme,
				logr.Discard(),
				objectmodifier.NewStageBatchModifierAll(k8sClient, scheme),
			)

			got, err := r.isLastStage(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
