package chain

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/externalsecrets"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/rbac"
)

const (
	externalSecretIntegrationRoleName   = "external-secret-integration"
	secretIntegrationServiceAccountName = "secret-manager"
	secretStoreName                     = "edp-system"
	externalSecretName                  = "regcred"
	manageSecretsEnv                    = "MANAGE_SECRETS"
)

// ConfigureManageSecretsRBAC is a stage chain element that configures RBAC for external secret integration.
type ConfigureManageSecretsRBAC struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
}

// ServeRequest implements the logic to configure RBAC for external secret integration.
func (h ConfigureManageSecretsRBAC) ServeRequest(stage *cdPipeApi.Stage) error {
	ctx := context.Background() // TODO: pass ctx from the caller
	logger := h.log.WithValues("stage", stage.Name, "target-ns", stage.Spec.Namespace)

	if !shouldManageSecretes() {
		logger.Info("Secrets management is disabled, skipping")
		return nextServeOrNil(h.next, stage)
	}

	logger.Info("Configuring RBAC for external secret integration")

	externalSecretIntegrationRole := &rbacApi.Role{}
	if err := h.client.Get(ctx, client.ObjectKey{
		Name:      "external-secret-integration",
		Namespace: stage.Namespace,
	}, externalSecretIntegrationRole); err != nil {
		return fmt.Errorf("failed to get %s role: %w", "external-secret-integration", err)
	}

	serviceAccount, err := h.createServiceAccount(ctrl.LoggerInto(ctx, logger), stage.Spec.Namespace)
	if err != nil {
		return err
	}

	if _, err = h.createRoleBinding(
		ctrl.LoggerInto(ctx, logger),
		stage.Namespace,
		stage.Spec.Namespace,
		serviceAccount.Name,
		externalSecretIntegrationRole.Name,
	); err != nil {
		return err
	}

	secretStore, err := h.createSecretStore(ctrl.LoggerInto(ctx, logger), stage.Namespace, stage.Spec.Namespace, serviceAccount.Name)
	if err != nil {
		return err
	}

	if _, err = h.createExternalSecret(
		ctrl.LoggerInto(ctx, logger),
		stage.Spec.Namespace,
		secretStore.GetName(),
	); err != nil {
		return err
	}

	logger.Info("RBAC for external secret integration has been configured successfully")

	return nextServeOrNil(h.next, stage)
}

func (h ConfigureManageSecretsRBAC) createServiceAccount(ctx context.Context, namespace string) (*corev1.ServiceAccount, error) {
	l := ctrl.LoggerFrom(ctx)

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      secretIntegrationServiceAccountName,
			Namespace: namespace,
		},
	}
	if err := h.client.Create(ctx, serviceAccount); err != nil {
		if !k8sErrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create %s service account: %w", serviceAccount.Name, err)
		}

		l.Info("Service account for external secret integration already exists")
	}

	return serviceAccount, nil
}

func (h ConfigureManageSecretsRBAC) createRoleBinding(
	ctx context.Context,
	stageNamespace,
	stageTargetNamespace,
	serviceAccountName,
	roleName string,
) (*rbacApi.RoleBinding, error) {
	l := ctrl.LoggerFrom(ctx)

	secretManagerRoleBinding := &rbacApi.RoleBinding{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("eso-%s", stageTargetNamespace),
			Namespace: stageNamespace,
		},
		Subjects: []rbacApi.Subject{
			{
				Kind:      rbacApi.ServiceAccountKind,
				Name:      serviceAccountName,
				Namespace: stageTargetNamespace,
			},
		},
		RoleRef: rbacApi.RoleRef{
			APIGroup: rbacApi.GroupName,
			Kind:     rbac.RoleKind,
			Name:     roleName,
		},
	}

	if err := h.client.Create(ctx, secretManagerRoleBinding); err != nil {
		if !k8sErrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create %s rolebinding: %w", secretManagerRoleBinding.Name, err)
		}

		l.Info("RoleBinding for external secret integration already exists")
	}

	return secretManagerRoleBinding, nil
}

func (h ConfigureManageSecretsRBAC) createSecretStore(
	ctx context.Context,
	stageNamespace,
	stageTargetNamespace,
	serviceAccountName string,
) (*unstructured.Unstructured, error) {
	l := ctrl.LoggerFrom(ctx)

	secretStore := externalsecrets.NewSecretStore(secretStoreName, stageTargetNamespace)
	secretStore.Object["spec"] = map[string]interface{}{
		"provider": map[string]interface{}{
			"kubernetes": map[string]interface{}{
				"remoteNamespace": stageNamespace,
				"auth": map[string]interface{}{
					"serviceAccount": map[string]interface{}{
						"name": serviceAccountName,
					},
				},
				"server": map[string]interface{}{
					"caProvider": map[string]interface{}{
						"type": "ConfigMap",
						"name": "kube-root-ca.crt",
						"key":  "ca.crt",
					},
				},
			},
		},
	}

	if err := h.client.Create(ctx, secretStore); err != nil {
		if !k8sErrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create %s secret store: %w", secretStore.GetName(), err)
		}

		l.Info("Secret store for external secret integration already exists")
	}

	return secretStore, nil
}

func (h ConfigureManageSecretsRBAC) createExternalSecret(
	ctx context.Context,
	stageTargetNamespace,
	secretStoreName string,
) (*unstructured.Unstructured, error) {
	l := ctrl.LoggerFrom(ctx)

	externalSecret := externalsecrets.NewExternalSecret(externalSecretName, stageTargetNamespace)
	externalSecret.Object["spec"] = map[string]interface{}{
		"refreshInterval": "1h",
		"secretStoreRef": map[string]interface{}{
			"kind": externalsecrets.SecretStoreKind,
			"name": secretStoreName,
		},
		"data": []interface{}{
			map[string]interface{}{
				"secretKey": "secretValue",
				"remoteRef": map[string]interface{}{
					"key":                "regcred",
					"property":           ".dockerconfigjson",
					"decodingStrategy":   "None",
					"conversionStrategy": "Default",
				},
			},
		},
		"target": map[string]interface{}{
			"creationPolicy": "Owner",
			"deletionPolicy": "Retain",
			"template": map[string]interface{}{
				"engineVersion": "v2",
				"type":          "kubernetes.io/dockerconfigjson",
				"data": map[string]interface{}{
					".dockerconfigjson": "{{ .secretValue | toString }}",
				},
			},
		},
	}

	if err := h.client.Create(ctx, externalSecret); err != nil {
		if !k8sErrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create %s external secret: %w", externalSecret.GetName(), err)
		}

		l.Info("External secret for external secret integration already exists")
	}

	return externalSecret, nil
}

func shouldManageSecretes() bool {
	val, exists := os.LookupEnv(manageSecretsEnv)
	if !exists {
		return false
	}

	enabled, err := strconv.ParseBool(val)
	if err != nil {
		return false
	}

	return enabled
}
