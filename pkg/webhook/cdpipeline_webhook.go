package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

// +kubebuilder:webhook:path=/validate-v2-edp-epam-com-v1-cdpipeline,mutating=false,failurePolicy=fail,sideEffects=None,groups=v2.edp.epam.com,resources=cdpipelines,verbs=update;delete,versions=v1,name=cdpipeline.epam.com,admissionReviewVersions=v1

// CDPipelineValidationWebhook is a webhook for validating CDPipeline CRD.
type CDPipelineValidationWebhook struct {
	client client.Client
}

// NewCDPipelineValidationWebhook creates a new webhook for validating CDPipeline CR.
func NewCDPipelineValidationWebhook(k8sClient client.Client) *CDPipelineValidationWebhook {
	return &CDPipelineValidationWebhook{client: k8sClient}
}

// SetupWebhookWithManager sets up the webhook with the manager for CDPipeline CR.
func (r *CDPipelineValidationWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewWebhookManagedBy(mgr).
		For(&pipelineApi.CDPipeline{}).
		WithValidator(r).
		Complete()
	if err != nil {
		return fmt.Errorf("failed to build CDPipeline validation webhook: %w", err)
	}

	return nil
}

var _ webhook.CustomValidator = &CDPipelineValidationWebhook{}

// ValidateCreate is a webhook for validating the creation of the CDPipeline CR.
func (*CDPipelineValidationWebhook) ValidateCreate(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate is a webhook for validating the updating of the CDPipeline CR.
func (*CDPipelineValidationWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	pipe, ok := newObj.(*pipelineApi.CDPipeline)
	if !ok {
		return nil, nil
	}

	return nil, checkResourceProtectionFromModificationOnUpdate(oldObj, pipe)
}

// ValidateDelete is a webhook for validating the deleting of the CDPipeline CR.
func (*CDPipelineValidationWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, checkResourceProtectionFromModificationOnDelete(obj)
}
