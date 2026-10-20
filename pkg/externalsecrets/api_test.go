package externalsecrets_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/epam/edp-cd-pipeline-operator/v2/internal/esotest"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/externalsecrets"
)

type failingMapper struct {
	meta.RESTMapper
	err error
}

func (m failingMapper) RESTMapping(schema.GroupKind, ...string) (*meta.RESTMapping, error) {
	return nil, m.err
}

// esoDiscoveryServer serves API discovery for the ESO kinds in a set of versions that can change at runtime.
type esoDiscoveryServer struct {
	mu       sync.Mutex
	versions []string
}

func (s *esoDiscoveryServer) setVersions(versions ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.versions = versions
}

func (s *esoDiscoveryServer) routes() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	groupVersions := make([]metaV1.GroupVersionForDiscovery, 0, len(s.versions))
	for _, v := range s.versions {
		groupVersions = append(groupVersions, metaV1.GroupVersionForDiscovery{
			GroupVersion: externalsecrets.Group + "/" + v,
			Version:      v,
		})
	}

	groups := metaV1.APIGroupList{}
	if len(groupVersions) > 0 {
		groups.Groups = append(groups.Groups, metaV1.APIGroup{
			Name:             externalsecrets.Group,
			Versions:         groupVersions,
			PreferredVersion: groupVersions[0],
		})
	}

	routes := map[string]any{
		"/api":    metaV1.APIVersions{Versions: []string{"v1"}},
		"/api/v1": metaV1.APIResourceList{GroupVersion: "v1"},
		"/apis":   groups,
	}

	for _, v := range s.versions {
		routes["/apis/"+externalsecrets.Group+"/"+v] = metaV1.APIResourceList{
			GroupVersion: externalsecrets.Group + "/" + v,
			APIResources: []metaV1.APIResource{
				{Name: "secretstores", Namespaced: true, Kind: externalsecrets.SecretStoreKind},
				{Name: "externalsecrets", Namespaced: true, Kind: externalsecrets.ExternalSecretKind},
			},
		}
	}

	return routes
}

func (s *esoDiscoveryServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, ok := s.routes()[r.URL.Path]
	if !ok {
		http.NotFound(w, r)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

// newDiscoveryMapper returns the controller-runtime dynamic RESTMapper backed by a
// discovery server that serves the ESO kinds in the given versions.
func newDiscoveryMapper(t *testing.T, versions ...string) (meta.RESTMapper, *esoDiscoveryServer) {
	t.Helper()

	discovery := &esoDiscoveryServer{versions: versions}

	server := httptest.NewServer(discovery)
	t.Cleanup(server.Close)

	cfg := &rest.Config{Host: server.URL}

	httpClient, err := rest.HTTPClientFor(cfg)
	require.NoError(t, err)

	mapper, err := apiutil.NewDynamicRESTMapper(cfg, httpClient)
	require.NoError(t, err)

	return mapper, discovery
}

func discoveryMapper(versions ...string) func(t *testing.T) meta.RESTMapper {
	return func(t *testing.T) meta.RESTMapper {
		mapper, _ := newDiscoveryMapper(t, versions...)

		return mapper
	}
}

func TestResolveAPIVersion_ESOUpgradedInPlace(t *testing.T) {
	t.Parallel()

	mapper, discovery := newDiscoveryMapper(t, "v1alpha1", "v1beta1")

	got, err := externalsecrets.ResolveAPIVersion(mapper)
	require.NoError(t, err)
	assert.Equal(t, "external-secrets.io/v1beta1", got)

	discovery.setVersions("v1")

	got, err = externalsecrets.ResolveAPIVersion(mapper)
	require.NoError(t, err)
	assert.Equal(t, "external-secrets.io/v1", got)
}

func TestResolveAPIVersion(t *testing.T) {
	t.Parallel()

	discoveryErr := errors.New("connection refused")

	v1 := schema.GroupVersion{Group: externalsecrets.Group, Version: "v1"}
	v1beta1 := schema.GroupVersion{Group: externalsecrets.Group, Version: "v1beta1"}
	secretStoreAheadOfExternalSecret := meta.NewDefaultRESTMapper([]schema.GroupVersion{v1, v1beta1})
	secretStoreAheadOfExternalSecret.Add(v1.WithKind(externalsecrets.SecretStoreKind), meta.RESTScopeNamespace)
	secretStoreAheadOfExternalSecret.Add(v1beta1.WithKind(externalsecrets.SecretStoreKind), meta.RESTScopeNamespace)
	secretStoreAheadOfExternalSecret.Add(v1beta1.WithKind(externalsecrets.ExternalSecretKind), meta.RESTScopeNamespace)

	tests := []struct {
		name    string
		mapper  func(t *testing.T) meta.RESTMapper
		want    string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "only v1 served",
			mapper:  func(*testing.T) meta.RESTMapper { return esotest.NewRESTMapper("v1") },
			want:    "external-secrets.io/v1",
			wantErr: require.NoError,
		},
		{
			name:    "v1 and v1beta1 served, v1 preferred",
			mapper:  func(*testing.T) meta.RESTMapper { return esotest.NewRESTMapper("v1beta1", "v1") },
			want:    "external-secrets.io/v1",
			wantErr: require.NoError,
		},
		{
			name:    "only v1beta1 served",
			mapper:  func(*testing.T) meta.RESTMapper { return esotest.NewRESTMapper("v1beta1") },
			want:    "external-secrets.io/v1beta1",
			wantErr: require.NoError,
		},
		{
			name:    "kinds served at different versions, common version used",
			mapper:  func(*testing.T) meta.RESTMapper { return secretStoreAheadOfExternalSecret },
			want:    "external-secrets.io/v1beta1",
			wantErr: require.NoError,
		},
		{
			name:   "no supported version served",
			mapper: func(*testing.T) meta.RESTMapper { return esotest.NewRESTMapper() },
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.ErrorIs(t, err, externalsecrets.ErrAPINotServed)
				require.ErrorContains(t, err, "v1, v1beta1")
			},
		},
		{
			name:   "unsupported version only",
			mapper: func(*testing.T) meta.RESTMapper { return esotest.NewRESTMapper("v1alpha1") },
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.ErrorIs(t, err, externalsecrets.ErrAPINotServed)
			},
		},
		{
			name: "discovery failure is not treated as not served",
			mapper: func(*testing.T) meta.RESTMapper {
				return failingMapper{RESTMapper: esotest.NewRESTMapper(), err: discoveryErr}
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.ErrorIs(t, err, discoveryErr)
				require.NotErrorIs(t, err, externalsecrets.ErrAPINotServed)
			},
		},
		{
			name:    "discovery mapper, only v1 served",
			mapper:  discoveryMapper("v1"),
			want:    "external-secrets.io/v1",
			wantErr: require.NoError,
		},
		{
			name:    "discovery mapper, v1 and v1beta1 served",
			mapper:  discoveryMapper("v1", "v1beta1"),
			want:    "external-secrets.io/v1",
			wantErr: require.NoError,
		},
		{
			name:    "discovery mapper, only v1beta1 served",
			mapper:  discoveryMapper("v1beta1"),
			want:    "external-secrets.io/v1beta1",
			wantErr: require.NoError,
		},
		{
			name:   "discovery mapper, group not installed",
			mapper: discoveryMapper(),
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.ErrorIs(t, err, externalsecrets.ErrAPINotServed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := externalsecrets.ResolveAPIVersion(tt.mapper(t))

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewObjects(t *testing.T) {
	t.Parallel()

	store := externalsecrets.NewSecretStore("external-secrets.io/v1beta1", "store", "ns")
	assert.Equal(t, "external-secrets.io/v1beta1", store.GetAPIVersion())
	assert.Equal(t, externalsecrets.SecretStoreKind, store.GetKind())
	assert.Equal(t, "store", store.GetName())
	assert.Equal(t, "ns", store.GetNamespace())

	secret := externalsecrets.NewExternalSecret("external-secrets.io/v1", "secret", "ns")
	assert.Equal(t, "external-secrets.io/v1", secret.GetAPIVersion())
	assert.Equal(t, externalsecrets.ExternalSecretKind, secret.GetKind())
	assert.Equal(t, "secret", secret.GetName())
	assert.Equal(t, "ns", secret.GetNamespace())
}
