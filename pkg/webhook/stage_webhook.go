package webhook

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
)

const listLimit = 1000

// +kubebuilder:webhook:path=/validate-v2-edp-epam-com-v1-stage,mutating=false,failurePolicy=fail,sideEffects=None,groups=v2.edp.epam.com,resources=stages,verbs=create;update;delete,versions=v1,name=stage.epam.com,admissionReviewVersions=v1
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// StageValidationWebhook is a webhook for validating Stage CRD.
type StageValidationWebhook struct {
	client client.Client
}

// NewStageValidationWebhook creates a new webhook for validating Stage CR.
func NewStageValidationWebhook(k8sClient client.Client) *StageValidationWebhook {
	return &StageValidationWebhook{client: k8sClient}
}

// SetupWebhookWithManager sets up the webhook with the manager for Stage CR.
func (r *StageValidationWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&pipelineApi.Stage{}).
		WithValidator(r).
		Complete()
	if err != nil {
		return fmt.Errorf("failed to build Stage validation webhook: %w", err)
	}

	return nil
}

var _ webhook.CustomValidator = &StageValidationWebhook{}

// ValidateCreate is a webhook for validating the creation of the Stage CR.
func (r *StageValidationWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	createdStage, ok := obj.(*pipelineApi.Stage)
	if !ok {
		return nil, errors.New("the wrong object given, expected Stage")
	}

	if err := r.uniqueTargetNamespaces(ctx, createdStage); err != nil {
		return nil, err
	}

	return nil, r.uniqueTargetNamespaceAcrossCluster(ctx, createdStage)
}

// ValidateUpdate is a webhook for validating the updating of the Stage CR.
func (*StageValidationWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return nil, checkResourceProtectionFromModificationOnUpdate(oldObj, newObj)
}

// ValidateDelete is a webhook for validating the deleting of the Stage CR.
func (*StageValidationWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, checkResourceProtectionFromModificationOnDelete(obj)
}

func (r *StageValidationWebhook) uniqueTargetNamespaces(ctx context.Context, stage *pipelineApi.Stage) error {
	stages := &pipelineApi.StageList{}

	if err := r.client.List(
		ctx,
		stages,
		client.InNamespace(stage.Namespace),
		client.Limit(listLimit),
	); err != nil {
		return fmt.Errorf("failed to list stages: %w", err)
	}

	for i := range stages.Items {
		if stages.Items[i].Name == stage.Name || stages.Items[i].Spec.ClusterName != stage.Spec.ClusterName {
			continue
		}

		if stages.Items[i].Spec.Namespace == stage.Spec.Namespace {
			return fmt.Errorf(
				"namespace %s is already used in CDPipeline %s Stage %s",
				stage.Spec.Namespace,
				stages.Items[i].Spec.CdPipeline,
				stages.Items[i].Name,
			)
		}
	}

	return nil
}

func (r *StageValidationWebhook) uniqueTargetNamespaceAcrossCluster(ctx context.Context, stage *pipelineApi.Stage) error {
	namespaces := &corev1.NamespaceList{}
	if err := r.client.List(
		ctx,
		namespaces,
		client.MatchingLabels{
			util.TenantLabelName: stage.Spec.Namespace,
		},
	); err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	for i := range namespaces.Items {
		if namespaces.Items[i].Name == stage.Spec.Namespace {
			return fmt.Errorf("namespace %s is already used in the cluster", stage.Spec.Namespace)
		}
	}

	return nil
}
