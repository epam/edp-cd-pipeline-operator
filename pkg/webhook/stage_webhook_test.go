package webhook

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
)

// newStageWithOrder returns an in-cluster Stage whose target namespace is "<name>-ns".
func newStageWithOrder(namespace, name, cdPipeline string, order int) *pipelineApi.Stage {
	return &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: pipelineApi.StageSpec{
			Name:        name,
			CdPipeline:  cdPipeline,
			ClusterName: pipelineApi.InCluster,
			Namespace:   name + "-ns",
			Order:       order,
		},
	}
}

func newTerminatingStage(namespace, name, cdPipeline string, order int) *pipelineApi.Stage {
	stage := newStageWithOrder(namespace, name, cdPipeline, order)
	stage.Finalizers = []string{"test-finalizer"}
	stage.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	return stage
}

var listFails = interceptor.Funcs{
	List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
		return errors.New("connection refused")
	},
}

func errContains(msg string) require.ErrorAssertionFunc {
	return func(t require.TestingT, err error, _ ...interface{}) {
		require.ErrorContains(t, err, msg)
	}
}

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
						Order:       1,
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
						Order:       2,
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
						Order:       1,
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
			name: "namespace conflict is reported before order conflict",
			obj:  newStageWithOrder("default", "stage1", "pipeline", 0),
			client: func(t *testing.T) client.Client {
				stageWithSameTargetNsAndOrder := newStageWithOrder("default", "stage2", "pipeline", 0)
				stageWithSameTargetNsAndOrder.Spec.Namespace = "stage1-ns"

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					stageWithSameTargetNsAndOrder,
				).Build()
			},
			wantErr: errContains("namespace stage1-ns is already used in CDPipeline pipeline Stage stage2"),
		},
		{
			name: "order conflict is reported before the target namespace ownership",
			obj:  newStageWithOrder("default", "stage1", "pipeline", 1),
			client: func(t *testing.T) client.Client {
				unlabeledTargetNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "stage1-ns"}}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					newStageWithOrder("default", "stage2", "pipeline", 1),
					unlabeledTargetNamespace,
				).Build()
			},
			wantErr: errContains("order 1 is already used in CDPipeline pipeline Stage stage2"),
		},
		{
			name: "order is used by a Stage of another CDPipeline that carries this CDPipeline label",
			obj:  newStageWithOrder("default", "stage1", "pipeline", 1),
			client: func(t *testing.T) client.Client {
				mislabeled := newStageWithOrder("default", "stage2", "other", 1)
				mislabeled.Labels = map[string]string{pipelineApi.StageCdPipelineLabelName: "pipeline"}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(mislabeled).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "order conflict, the existing Stage is being deleted below a live Stage",
			obj:  newStageWithOrder("default", "stage1", "pipeline", 1),
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					newTerminatingStage("default", "stage2", "pipeline", 1),
					newStageWithOrder("default", "stage3", "pipeline", 2),
				).Build()
			},
			wantErr: errContains("order 1 is already used in CDPipeline pipeline Stage stage2"),
		},
		{
			name: "order of the last Stage that is being deleted is admitted",
			obj:  newStageWithOrder("default", "stage1", "pipeline", 1),
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					newTerminatingStage("default", "stage2", "pipeline", 1),
				).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "unused order, contiguity is not checked",
			obj:  newStageWithOrder("default", "stage1", "pipeline", 5),
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					newStageWithOrder("default", "stage2", "pipeline", 0),
				).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "order is used in another namespace",
			obj:  newStageWithOrder("default", "stage1", "pipeline", 1),
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					newStageWithOrder("other", "stage2", "pipeline", 1),
				).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "order is used by a Stage that is being deleted below a live Stage in another namespace",
			obj:  newStageWithOrder("default", "stage1", "pipeline", 1),
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					newTerminatingStage("other", "stage2", "pipeline", 1),
					newStageWithOrder("other", "stage3", "pipeline", 2),
				).Build()
			},
			wantErr: require.NoError,
		},
		{
			name: "listing the Stages fails",
			obj:  newStageWithOrder("default", "stage1", "pipeline", 1),
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(listFails).Build()
			},
			wantErr: errContains("failed to list stages: connection refused"),
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewStageValidationWebhook(tt.client(t))
			w, err := r.ValidateCreate(context.Background(), tt.obj)

			assert.Nil(t, w)
			tt.wantErr(t, err)
		})
	}
}

func TestStageValidationWebhook_ValidateCreate_TargetNamespaceOwnership(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, pipelineApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	const unreachableKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: remote
  cluster:
    server: https://127.0.0.1:1
contexts:
- name: remote
  context:
    cluster: remote
    user: remote
current-context: remote
users:
- name: remote
  user:
    token: token
`

	newStage := func(clusterName string) *pipelineApi.Stage {
		return &pipelineApi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: "stage1", Namespace: "default"},
			Spec: pipelineApi.StageSpec{
				Name:        "stage1",
				CdPipeline:  "pipeline",
				ClusterName: clusterName,
				Namespace:   "target-ns",
			},
		}
	}

	targetNamespace := func(labels map[string]string) *corev1.Namespace {
		return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target-ns", Labels: labels}}
	}

	clusterSecret := func(config string) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "remote", Namespace: "default"},
			Data:       map[string][]byte{"config": []byte(config)},
		}
	}

	tests := []struct {
		name         string
		capsule      bool
		openshift    bool
		stage        *pipelineApi.Stage
		objects      []client.Object
		wantNotOwned bool
		wantWarnings int
	}{
		{
			name:  "target namespace absent",
			stage: newStage(pipelineApi.InCluster),
		},
		{
			name:    "target namespace owned",
			stage:   newStage(pipelineApi.InCluster),
			objects: []client.Object{targetNamespace(map[string]string{util.TenantLabelName: "default"})},
		},
		{
			name:         "target namespace unlabeled",
			stage:        newStage(pipelineApi.InCluster),
			objects:      []client.Object{targetNamespace(nil)},
			wantNotOwned: true,
		},
		{
			name:         "target namespace labeled for another tenant",
			stage:        newStage(pipelineApi.InCluster),
			objects:      []client.Object{targetNamespace(map[string]string{util.TenantLabelName: "other"})},
			wantNotOwned: true,
		},
		{
			name:    "capsule in-cluster skips the check",
			capsule: true,
			stage:   newStage(pipelineApi.InCluster),
			objects: []client.Object{targetNamespace(nil)},
		},
		{
			name:      "openshift skips the check",
			openshift: true,
			stage:     newStage(pipelineApi.InCluster),
			objects:   []client.Object{targetNamespace(nil)},
		},
		{
			name:         "remote cluster unreachable warns",
			stage:        newStage("remote"),
			objects:      []client.Object{clusterSecret(unreachableKubeconfig)},
			wantWarnings: 1,
		},
		{
			name:         "remote cluster with invalid kubeconfig warns",
			stage:        newStage("remote"),
			objects:      []client.Object{clusterSecret("not a kubeconfig")},
			wantWarnings: 1,
		},
		{
			name:         "remote cluster without cluster secret warns",
			stage:        newStage("remote"),
			wantWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.capsule {
				t.Setenv("TENANCY_ENGINE", "capsule")
			}

			if tt.openshift {
				t.Setenv("PLATFORM_TYPE", "openshift")
			}

			r := NewStageValidationWebhook(fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build())
			w, err := r.ValidateCreate(context.Background(), tt.stage)

			if tt.wantNotOwned {
				require.ErrorIs(t, err, util.ErrNamespaceNotOwned)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, w, tt.wantWarnings)

			for _, warning := range w {
				assert.Equal(t,
					"cannot verify target namespace ownership on cluster "+tt.stage.Spec.ClusterName+"; the operator re-checks",
					warning,
				)
			}
		})
	}
}

type blockingGetClient struct {
	client.Client
}

func (blockingGetClient) Get(ctx context.Context, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
	<-ctx.Done()

	return ctx.Err()
}

func TestStageValidationWebhook_ValidateCreate_LookupDeadlineWarns(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, pipelineApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	stage := &pipelineApi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "stage1", Namespace: "default"},
		Spec: pipelineApi.StageSpec{
			Name:        "stage1",
			CdPipeline:  "pipeline",
			ClusterName: pipelineApi.InCluster,
			Namespace:   "target-ns",
		},
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	r := NewStageValidationWebhook(blockingGetClient{Client: fake.NewClientBuilder().WithScheme(scheme).Build()})
	w, err := r.ValidateCreate(ctx, stage)

	require.NoError(t, err)
	require.Len(t, w, 1)
	assert.NotContains(t, w[0], context.DeadlineExceeded.Error())
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
			w, err := cd.ValidateUpdate(context.Background(), tt.args.oldObj, tt.args.newObj)

			assert.Nil(t, w)
			tt.wantErr(t, err)
		})
	}
}

func TestStageValidationWebhook_ValidateUpdate_Order(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, pipelineApi.AddToScheme(scheme))

	stageBeforeFinalizerRemoval := newStageWithOrder("default", "stage2", "pipeline", 1)
	stageBeforeFinalizerRemoval.Finalizers = []string{"test-finalizer"}

	tests := []struct {
		name    string
		objects []client.Object
		funcs   interceptor.Funcs
		oldObj  runtime.Object
		newObj  runtime.Object
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "order changed to a free value",
			objects: []client.Object{
				newStageWithOrder("default", "stage1", "pipeline", 0),
				newStageWithOrder("default", "stage2", "pipeline", 1),
			},
			oldObj:  newStageWithOrder("default", "stage2", "pipeline", 1),
			newObj:  newStageWithOrder("default", "stage2", "pipeline", 3),
			wantErr: require.NoError,
		},
		{
			name: "order changed to a used value",
			objects: []client.Object{
				newStageWithOrder("default", "stage1", "pipeline", 0),
				newStageWithOrder("default", "stage2", "pipeline", 1),
			},
			oldObj:  newStageWithOrder("default", "stage2", "pipeline", 1),
			newObj:  newStageWithOrder("default", "stage2", "pipeline", 0),
			wantErr: errContains("order 0 is already used in CDPipeline pipeline Stage stage1"),
		},
		{
			name: "order and CDPipeline unchanged while a duplicate exists",
			objects: []client.Object{
				newStageWithOrder("default", "stage1", "pipeline", 1),
				newStageWithOrder("default", "stage2", "pipeline", 1),
			},
			oldObj:  stageBeforeFinalizerRemoval,
			newObj:  newStageWithOrder("default", "stage2", "pipeline", 1),
			wantErr: require.NoError,
		},
		{
			name: "CDPipeline changed to one where the order is taken",
			objects: []client.Object{
				newStageWithOrder("default", "stage1", "other-pipeline", 1),
				newStageWithOrder("default", "stage2", "pipeline", 1),
			},
			oldObj:  newStageWithOrder("default", "stage2", "pipeline", 1),
			newObj:  newStageWithOrder("default", "stage2", "other-pipeline", 1),
			wantErr: errContains("order 1 is already used in CDPipeline other-pipeline Stage stage1"),
		},
		{
			name: "CDPipeline changed to one where the order is free",
			objects: []client.Object{
				newStageWithOrder("default", "stage1", "other-pipeline", 0),
				newStageWithOrder("default", "stage2", "pipeline", 1),
			},
			oldObj:  newStageWithOrder("default", "stage2", "pipeline", 1),
			newObj:  newStageWithOrder("default", "stage2", "other-pipeline", 1),
			wantErr: require.NoError,
		},
		{
			name:    "listing the Stages fails",
			funcs:   listFails,
			oldObj:  newStageWithOrder("default", "stage2", "pipeline", 1),
			newObj:  newStageWithOrder("default", "stage2", "pipeline", 2),
			wantErr: errContains("failed to list stages: connection refused"),
		},
		{
			name:    "invalid object given",
			oldObj:  &codebaseApi.Codebase{},
			newObj:  &codebaseApi.Codebase{},
			wantErr: errContains("the wrong object given, expected Stage"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewStageValidationWebhook(
				fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).WithInterceptorFuncs(tt.funcs).Build(),
			)
			w, err := r.ValidateUpdate(context.Background(), tt.oldObj, tt.newObj)

			assert.Nil(t, w)
			tt.wantErr(t, err)
		})
	}
}

func TestUniqueOrder(t *testing.T) {
	t.Parallel()

	stageList := func(stages ...*pipelineApi.Stage) *pipelineApi.StageList {
		items := make([]pipelineApi.Stage, 0, len(stages))

		for _, stage := range stages {
			items = append(items, *stage)
		}

		return &pipelineApi.StageList{Items: items}
	}

	tests := []struct {
		name    string
		stages  *pipelineApi.StageList
		stage   *pipelineApi.Stage
		wantErr string
	}{
		{
			name:    "order is used by a live Stage",
			stages:  stageList(newStageWithOrder("default", "stage2", "pipeline", 1)),
			stage:   newStageWithOrder("default", "stage1", "pipeline", 1),
			wantErr: "order 1 is already used in CDPipeline pipeline Stage stage2",
		},
		{
			name:   "order is used in another CDPipeline",
			stages: stageList(newStageWithOrder("default", "stage2", "other-pipeline", 1)),
			stage:  newStageWithOrder("default", "stage1", "pipeline", 1),
		},
		{
			name:   "order is used by the Stage itself",
			stages: stageList(newStageWithOrder("default", "stage1", "pipeline", 1)),
			stage:  newStageWithOrder("default", "stage1", "pipeline", 1),
		},
		{
			name:   "order is used by the last Stage that is being deleted",
			stages: stageList(newTerminatingStage("default", "stage2", "pipeline", 1)),
			stage:  newStageWithOrder("default", "stage1", "pipeline", 1),
		},
		{
			name: "order is used by a Stage that is being deleted below a live Stage",
			stages: stageList(
				newTerminatingStage("default", "stage2", "pipeline", 1),
				newStageWithOrder("default", "stage3", "pipeline", 2),
			),
			stage:   newStageWithOrder("default", "stage1", "pipeline", 1),
			wantErr: "order 1 is already used in CDPipeline pipeline Stage stage2",
		},
		{
			name: "order is used by a Stage that is being deleted below Stages that are being deleted",
			stages: stageList(
				newTerminatingStage("default", "stage2", "pipeline", 1),
				newTerminatingStage("default", "stage3", "pipeline", 2),
			),
			stage:   newStageWithOrder("default", "stage1", "pipeline", 1),
			wantErr: "order 1 is already used in CDPipeline pipeline Stage stage2",
		},
		{
			name: "order is used by a Stage that is being deleted below a live Stage of another CDPipeline",
			stages: stageList(
				newTerminatingStage("default", "stage2", "pipeline", 1),
				newStageWithOrder("default", "stage3", "other-pipeline", 2),
			),
			stage: newStageWithOrder("default", "stage1", "pipeline", 1),
		},
		{
			name: "order is used by a Stage that is being deleted below a live Stage in another CDPipeline",
			stages: stageList(
				newTerminatingStage("default", "stage2", "other-pipeline", 1),
				newStageWithOrder("default", "stage3", "other-pipeline", 2),
			),
			stage: newStageWithOrder("default", "stage1", "pipeline", 1),
		},
		{
			name: "order is used by a Stage that is being deleted below the Stage under validation",
			stages: stageList(
				newTerminatingStage("default", "stage2", "pipeline", 1),
				newStageWithOrder("default", "stage1", "pipeline", 5),
			),
			stage: newStageWithOrder("default", "stage1", "pipeline", 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := uniqueOrder(tt.stages, tt.stage)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
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
			w, err := st.ValidateDelete(context.Background(), tt.args.obj)

			assert.Nil(t, w)
			tt.wantErr(t, err)
		})
	}
}
