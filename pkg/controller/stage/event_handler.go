package stage

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
)

// PipelineEventHandler is a handler for CDPipeline events,
// which triggers all of its stages of reconciliation.
// It only triggers stages on the CDPipeline update event,
// as only in this case can we add new applications to the pipeline.
type PipelineEventHandler struct {
	client client.Client
	log    logr.Logger
}

// NewPipelineEventHandler creates a new PipelineEventHandler.
func NewPipelineEventHandler(client client.Client, log logr.Logger) *PipelineEventHandler {
	return &PipelineEventHandler{client: client, log: log}
}

// Update triggers all stages of the CDPipeline reconciliation.
func (h *PipelineEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectNew == nil {
		h.log.Info("Object is nil")
		return
	}

	_, ok := evt.ObjectNew.(*cdPipeApi.CDPipeline)
	if !ok {
		h.log.Info("Object is not CDPipeline")
		return
	}

	stages := &cdPipeApi.StageList{}
	if err := h.client.List(
		context.Background(),
		stages,
		client.InNamespace(evt.ObjectNew.GetNamespace()),
		client.MatchingLabels{cdPipeApi.CodebaseTypeLabelName: evt.ObjectNew.GetName()},
		client.Limit(1000),
	); err != nil {
		h.log.Error(err, "unable to get stages for cd pipeline", "cd pipeline", evt.ObjectNew.GetName())
		return
	}

	for _, stage := range stages.Items {
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: stage.GetNamespace(),
			Name:      stage.GetName(),
		}})
	}
}

// Create does nothing, skip event.
func (h *PipelineEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
}

// Delete does nothing, skip event.
func (h *PipelineEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
}

// Generic does nothing, skip event.
func (h *PipelineEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
}
