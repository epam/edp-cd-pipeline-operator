package externalsecrets

import (
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// Group is the External Secrets Operator API group.
	Group              = "external-secrets.io"
	SecretStoreKind    = "SecretStore"
	ExternalSecretKind = "ExternalSecret"
)

// supportedVersions lists the ESO API versions in order of preference.
// ESO 0.17+ serves only v1, 0.16 serves both, 0.15 and older serve only v1beta1.
var supportedVersions = [...]string{"v1", "v1beta1"}

// ErrAPINotServed is returned when the cluster serves none of the supported versions for every ESO kind.
var ErrAPINotServed = errors.New("external secrets operator is not installed or does not serve a supported API version")

// ResolveAPIVersion returns the apiVersion (group/version) of the most preferred supported version
// in which the cluster behind mapper serves both SecretStore and ExternalSecret.
// Each version is probed explicitly; a version missing from the mapper cache triggers discovery.
// A newly served version is picked up without an operator restart. A version that stops being
// served stays cached until the operator restarts.
func ResolveAPIVersion(mapper meta.RESTMapper) (string, error) {
	for _, version := range supportedVersions {
		served, err := servedInVersion(mapper, version)
		if err != nil {
			return "", err
		}

		if served {
			return schema.GroupVersion{Group: Group, Version: version}.String(), nil
		}
	}

	return "", fmt.Errorf("%w: %s %s and %s are not served together in any of versions %s",
		ErrAPINotServed, Group, SecretStoreKind, ExternalSecretKind, strings.Join(supportedVersions[:], ", "))
}

func servedInVersion(mapper meta.RESTMapper, version string) (bool, error) {
	for _, kind := range []string{SecretStoreKind, ExternalSecretKind} {
		if _, err := mapper.RESTMapping(schema.GroupKind{Group: Group, Kind: kind}, version); err != nil {
			if meta.IsNoMatchError(err) {
				return false, nil
			}

			return false, fmt.Errorf("failed to resolve %s %s API version %s: %w", Group, kind, version, err)
		}
	}

	return true, nil
}

func NewSecretStore(apiVersion, name, namespace string) *unstructured.Unstructured {
	return newObject(apiVersion, SecretStoreKind, name, namespace)
}

func NewExternalSecret(apiVersion, name, namespace string) *unstructured.Unstructured {
	return newObject(apiVersion, ExternalSecretKind, name, namespace)
}

func newObject(apiVersion, kind, name, namespace string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.Object = map[string]interface{}{
		"kind":       kind,
		"apiVersion": apiVersion,
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
	}

	return obj
}
