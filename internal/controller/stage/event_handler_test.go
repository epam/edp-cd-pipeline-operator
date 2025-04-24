package stage

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

func TestPipelineEventHandler_Update(t *testing.T) {
	scheme := runtime.NewScheme()
	err := cdPipeApi.AddToScheme(scheme)
	require.NoError(t, err)

	tests := []struct {
		name    string
		evt     event.UpdateEvent
		objects []client.Object
		expLen  int
	}{
		{
			name: "should add request to queue",
			evt: event.UpdateEvent{
				ObjectNew: &cdPipeApi.CDPipeline{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "default",
						Name:      "cd-pipeline",
					},
				},
			},
			objects: []client.Object{
				&cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "default",
						Name:      "stage",
						Labels: map[string]string{
							cdPipeApi.StageCdPipelineLabelName: "cd-pipeline",
						},
					},
				},
			},
			expLen: 1,
		},
		{
			name: "should skip one stage without pipeline label",
			evt: event.UpdateEvent{
				ObjectNew: &cdPipeApi.CDPipeline{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "default",
						Name:      "cd-pipeline",
					},
				},
			},
			objects: []client.Object{
				&cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "default",
						Name:      "stage",
						Labels: map[string]string{
							cdPipeApi.StageCdPipelineLabelName: "cd-pipeline",
						},
					},
				},
				&cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "default",
						Name:      "stage2",
					},
				},
			},
			expLen: 1,
		},
		{
			name: "should skip one stage in another namespace",
			evt: event.UpdateEvent{
				ObjectNew: &cdPipeApi.CDPipeline{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "default",
						Name:      "cd-pipeline",
					},
				},
			},
			objects: []client.Object{
				&cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "default",
						Name:      "stage",
						Labels: map[string]string{
							cdPipeApi.StageCdPipelineLabelName: "cd-pipeline",
						},
					},
				},
				&cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "dev",
						Name:      "stage2",
						Labels: map[string]string{
							cdPipeApi.StageCdPipelineLabelName: "cd-pipeline",
						},
					},
				},
			},
			expLen: 1,
		},
		{
			name:   "empty update event object",
			evt:    event.UpdateEvent{},
			expLen: 0,
		},
		{
			name: "event object with invalid kind",
			evt: event.UpdateEvent{
				ObjectNew: &cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "default",
						Name:      "stage",
					},
				},
			},
			objects: []client.Object{
				&cdPipeApi.Stage{
					ObjectMeta: metaV1.ObjectMeta{
						Namespace: "default",
						Name:      "stage",
						Labels: map[string]string{
							cdPipeApi.StageCdPipelineLabelName: "cd-pipeline",
						},
					},
				},
			},
			expLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewPipelineEventHandler(
				fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build(),
				logr.Discard(),
			)

			q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

			h.Update(t.Context(), tt.evt, q)

			assert.Equal(t, tt.expLen, q.Len())
		})
	}
}

func TestPipelineEventHandler_Create(t *testing.T) {
	scheme := runtime.NewScheme()
	err := cdPipeApi.AddToScheme(scheme)
	require.NoError(t, err)

	h := NewPipelineEventHandler(
		fake.NewClientBuilder().WithScheme(scheme).Build(),
		logr.Discard(),
	)

	q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

	h.Create(t.Context(), event.CreateEvent{}, q)

	assert.Equal(t, 0, q.Len())
}

func TestPipelineEventHandler_Delete(t *testing.T) {
	scheme := runtime.NewScheme()
	err := cdPipeApi.AddToScheme(scheme)
	require.NoError(t, err)

	h := NewPipelineEventHandler(
		fake.NewClientBuilder().WithScheme(scheme).Build(),
		logr.Discard(),
	)

	q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

	h.Delete(t.Context(), event.DeleteEvent{}, q)

	assert.Equal(t, 0, q.Len())
}

func TestPipelineEventHandler_Generic(t *testing.T) {
	scheme := runtime.NewScheme()
	err := cdPipeApi.AddToScheme(scheme)
	require.NoError(t, err)

	h := NewPipelineEventHandler(
		fake.NewClientBuilder().WithScheme(scheme).Build(),
		logr.Discard(),
	)

	q := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

	h.Generic(t.Context(), event.GenericEvent{}, q)

	assert.Equal(t, 0, q.Len())
}
