package webhook

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
)

// RegisterValidationWebHook registers a new webhook for validating CRD.
func RegisterValidationWebHook(mgr ctrl.Manager) error {
	if err := NewStageValidationWebhook(mgr.GetClient()).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("failed to create Stage webhook: %w", err)
	}

	if err := NewCDPipelineValidationWebhook(mgr.GetClient()).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("failed to create CDpipeline webhook: %w", err)
	}

	return nil
}
