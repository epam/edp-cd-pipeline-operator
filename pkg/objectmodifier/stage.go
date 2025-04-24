package objectmodifier

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/helper"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

// StageModifier is an interface for modifying stage object.
type StageModifier interface {
	Apply(ctx context.Context, stage *cdPipeApi.Stage) (bool, error)
}

// StageModifierFunc is a function that implements StageModifier interface.
type StageModifierFunc func(ctx context.Context, stage *cdPipeApi.Stage) (bool, error)

// Apply implements StageModifier interface.
func (f StageModifierFunc) Apply(ctx context.Context, stage *cdPipeApi.Stage) (bool, error) {
	return f(ctx, stage)
}

// StageBatchModifier is a modifier that applies a list of modifiers.
type StageBatchModifier struct {
	k8sClient client.Writer
	modifiers []StageModifier
}

// NewStageBatchModifier returns a new instance of StageBatchModifier.
func NewStageBatchModifier(k8sClient client.Client, modifiers []StageModifier) *StageBatchModifier {
	return &StageBatchModifier{k8sClient: k8sClient, modifiers: modifiers}
}

// NewStageBatchModifierAll returns a new instance of StageBatchModifier with all the modifiers.
func NewStageBatchModifierAll(k8sClient client.Client, scheme *runtime.Scheme) *StageBatchModifier {
	modifiers := []StageModifier{
		StageModifierFunc(setStageLabel),
		newStageOwnerRefModifier(k8sClient, scheme),
	}

	return &StageBatchModifier{k8sClient: k8sClient, modifiers: modifiers}
}

// Apply applies all the modifiers to the stage.
func (m *StageBatchModifier) Apply(ctx context.Context, stage *cdPipeApi.Stage) (bool, error) {
	patch := client.MergeFrom(stage.DeepCopy())
	needToPatch := false

	for _, modifier := range m.modifiers {
		changed, err := modifier.Apply(ctx, stage)
		if err != nil {
			return false, fmt.Errorf("failed to apply modifier: %w", err)
		}

		if changed {
			needToPatch = true
		}
	}

	if needToPatch {
		if err := m.k8sClient.Patch(ctx, stage, patch); err != nil {
			return false, fmt.Errorf("failed to patch stage: %w", err)
		}

		return true, nil
	}

	return false, nil
}

// setStageLabel sets label to stage object.
func setStageLabel(ctx context.Context, stage *cdPipeApi.Stage) (bool, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Trying to update labels for stage")

	if stage == nil {
		return false, errors.New("failed to update stage labels: stage is nil")
	}

	labels := stage.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	if _, ok := labels[cdPipeApi.StageCdPipelineLabelName]; ok {
		log.Info("Stage already has label", "label", cdPipeApi.StageCdPipelineLabelName)
		return false, nil
	}

	labels[cdPipeApi.StageCdPipelineLabelName] = stage.Spec.CdPipeline
	stage.SetLabels(labels)

	log.Info("Stage labels were updated", "labels", labels)

	return true, nil
}

// stageOwnerRefModifier sets CDPipeline owner reference to stage object.
type stageOwnerRefModifier struct {
	k8sClient client.Client
	scheme    *runtime.Scheme
}

// newStageOwnerRefModifier returns a new instance of stageOwnerRefModifier.
func newStageOwnerRefModifier(k8sClient client.Client, scheme *runtime.Scheme) *stageOwnerRefModifier {
	return &stageOwnerRefModifier{k8sClient: k8sClient, scheme: scheme}
}

// Apply sets CDPipeline owner reference to stage object.
func (m *stageOwnerRefModifier) Apply(ctx context.Context, stage *cdPipeApi.Stage) (bool, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Trying to update owner reference for stage")

	if stage == nil {
		return false, errors.New("failed to update stage owner reference: stage is nil")
	}

	if ow := helper.GetOwnerReference(consts.CDPipelineKind, stage.GetOwnerReferences()); ow != nil {
		log.Info("CDPipeline owner reference already exists")
		return false, nil
	}

	pipeline := &cdPipeApi.CDPipeline{}
	if err := m.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: stage.Namespace,
		Name:      stage.Spec.CdPipeline,
	}, pipeline); err != nil {
		return false, fmt.Errorf("cdpipeline %s doesn't exist: %w", stage.Spec.CdPipeline, err)
	}

	if err := controllerutil.SetControllerReference(pipeline, stage, m.scheme); err != nil {
		return false, fmt.Errorf("couldn't set CDPipeline %s owner ref: %w", stage.Spec.CdPipeline, err)
	}

	log.Info("CDPipeline owner reference was updated")

	return true, nil
}
