package webhook

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
)

// RegisterValidationWebHook registers a new webhook for validating CRD.
func RegisterValidationWebHook(ctx context.Context, mgr ctrl.Manager, namespace string) error {
	// mgr.GetAPIReader() is used to read objects before cache is started.
	certService := NewCertService(mgr.GetAPIReader(), mgr.GetClient())
	if err := certService.PopulateCertificates(ctx, namespace); err != nil {
		return fmt.Errorf("failed to populate certificates: %w", err)
	}

	if err := NewStageValidationWebhook(mgr.GetClient()).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("failed to create Stage webhook: %w", err)
	}

	if err := NewCDPipelineValidationWebhook(mgr.GetClient()).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("failed to create CDpipeline webhook: %w", err)
	}

	return nil
}
