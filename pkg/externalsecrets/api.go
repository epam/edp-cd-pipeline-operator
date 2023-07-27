package externalsecrets

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

const (
	SecretStoreKind    = "SecretStore"
	ExternalSecretKind = "ExternalSecret"
	ApiVersion         = "external-secrets.io/v1beta1"
)

func NewSecretStore(name, namespace string) *unstructured.Unstructured {
	store := &unstructured.Unstructured{}
	store.Object = map[string]interface{}{
		"kind":       SecretStoreKind,
		"apiVersion": ApiVersion,
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
	}

	return store
}

func NewExternalSecret(name, namespace string) *unstructured.Unstructured {
	secret := &unstructured.Unstructured{}
	secret.Object = map[string]interface{}{
		"kind":       ExternalSecretKind,
		"apiVersion": ApiVersion,
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
	}

	return secret
}
