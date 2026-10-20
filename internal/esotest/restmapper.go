// Package esotest provides test helpers for code that depends on External Secrets Operator APIs.
package esotest

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/externalsecrets"
)

// NewRESTMapper returns a RESTMapper serving SecretStore and ExternalSecret in the given versions.
func NewRESTMapper(versions ...string) *meta.DefaultRESTMapper {
	gvs := make([]schema.GroupVersion, 0, len(versions))
	for _, v := range versions {
		gvs = append(gvs, schema.GroupVersion{Group: externalsecrets.Group, Version: v})
	}

	mapper := meta.NewDefaultRESTMapper(gvs)

	for _, gv := range gvs {
		mapper.Add(gv.WithKind(externalsecrets.SecretStoreKind), meta.RESTScopeNamespace)
		mapper.Add(gv.WithKind(externalsecrets.ExternalSecretKind), meta.RESTScopeNamespace)
	}

	return mapper
}
