package webhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/multiclusterclient"
)

const listLimit = 1000

// targetNamespaceLookupTimeout bounds the target namespace lookup so that an unreachable
// cluster yields a warning before the webhook own timeout (10s) expires.
const targetNamespaceLookupTimeout = 3 * time.Second

// +kubebuilder:webhook:path=/validate-v2-edp-epam-com-v1-stage,mutating=false,failurePolicy=fail,sideEffects=None,groups=v2.edp.epam.com,resources=stages,verbs=create;update;delete,versions=v1,name=stage.epam.com,admissionReviewVersions=v1
// Stage target namespaces are arbitrary, so this rule is cluster-scoped.
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get

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

	return r.targetNamespaceOwnership(ctx, createdStage)
}

// ValidateUpdate is a webhook for validating the updating of the Stage CR.
func (*StageValidationWebhook) ValidateUpdate(
	_ context.Context,
	oldObj, newObj runtime.Object,
) (admission.Warnings, error) {
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

// targetNamespaceOwnership rejects a Stage whose target namespace exists but is not owned by the Stage tenant.
// A target cluster that cannot be reached does not block the Stage, a warning is returned instead.
func (r *StageValidationWebhook) targetNamespaceOwnership(
	ctx context.Context,
	stage *pipelineApi.Stage,
) (admission.Warnings, error) {
	if !util.OwnershipEnforced(stage) {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, targetNamespaceLookupTimeout)
	defer cancel()

	targetClient, err := multiclusterclient.NewClientProvider(r.client).
		GetClusterClient(ctx, stage.Namespace, stage.Spec.ClusterName, client.Options{})
	if err != nil {
		return ownershipWarning(ctx, stage, err), nil
	}

	ns := &corev1.Namespace{}
	if err = targetClient.Get(ctx, client.ObjectKey{Name: stage.Spec.Namespace}, ns); err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}

		return ownershipWarning(ctx, stage, err), nil
	}

	return nil, util.CheckNamespaceOwnership(ns, stage.Namespace)
}

// ownershipWarning keeps the lookup error out of the response: it can reveal whether a cluster
// Secret exists and remote server URLs. The error is logged instead.
func ownershipWarning(ctx context.Context, stage *pipelineApi.Stage, err error) admission.Warnings {
	ctrl.LoggerFrom(ctx).Error(err, "Cannot verify target namespace ownership",
		"cluster", stage.Spec.ClusterName, "namespace", stage.Spec.Namespace)

	return admission.Warnings{
		fmt.Sprintf("cannot verify target namespace ownership on cluster %s; the operator re-checks", stage.Spec.ClusterName),
	}
}
