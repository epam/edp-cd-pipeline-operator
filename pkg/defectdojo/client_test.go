package defectdojo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testToken = "secret-token"

func newTestClient(t *testing.T, h http.HandlerFunc, opts ...Option) *Client {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	opts = append([]Option{WithRetry(2, time.Millisecond)}, opts...)

	return New(opts...).WithCredentials(srv.URL+"/", testToken)
}

func testRequest() ReimportRequest {
	return ReimportRequest{
		ProductType: "KubeRocketCI",
		Product:     "app",
		Engagement:  "runtime-cdp-dev",
		TestTitle:   "VulnerabilityReport deployment/app/app",
		Version:     "1.0.0",
		Tags:        []string{"krci-scope:runtime", "krci-env:cdp.dev"},
		ScanDate:    time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC),
		Report:      []byte(`{"kind":"VulnerabilityReport"}`),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestClient_Reimport_SendsMultipartFields(t *testing.T) {
	var (
		fields map[string][]string
		file   string
		auth   string
		path   string
	)

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		path = r.URL.Path

		require.NoError(t, r.ParseMultipartForm(1<<20))

		fields = r.MultipartForm.Value

		f, _, err := r.FormFile("file")
		require.NoError(t, err)

		b, err := io.ReadAll(f)
		require.NoError(t, err)

		file = string(b)

		writeJSON(w, http.StatusCreated, ReimportResult{TestID: 7, EngagementID: 8, ProductID: 9})
	})

	res, err := c.Reimport(context.Background(), testRequest())

	require.NoError(t, err)
	assert.Equal(t, ReimportResult{TestID: 7, EngagementID: 8, ProductID: 9}, res)
	assert.Equal(t, "Token "+testToken, auth)
	assert.Equal(t, "/api/v2/reimport-scan/", path)
	assert.JSONEq(t, `{"kind":"VulnerabilityReport"}`, file)
	assert.Equal(t, map[string][]string{
		"scan_type":                   {"Trivy Operator Scan"},
		"product_type_name":           {"KubeRocketCI"},
		"product_name":                {"app"},
		"engagement_name":             {"runtime-cdp-dev"},
		"test_title":                  {"VulnerabilityReport deployment/app/app"},
		"version":                     {"1.0.0"},
		"auto_create_context":         {"true"},
		"close_old_findings":          {"true"},
		"deduplication_on_engagement": {"true"},
		"active":                      {"true"},
		"verified":                    {"false"},
		"minimum_severity":            {"Info"},
		"environment":                 {"Development"},
		"push_to_jira":                {"false"},
		"scan_date":                   {"2026-11-10"},
		"tags":                        {"krci-scope:runtime", "krci-env:cdp.dev"},
	}, fields)
}

func TestClient_Reimport_Retries(t *testing.T) {
	tests := []struct {
		name         string
		statuses     []int
		wantAttempts int32
		wantStatus   int
		wantErr      bool
	}{
		{name: "500 then success", statuses: []int{500, 201}, wantAttempts: 2},
		{name: "429 then 503 then success", statuses: []int{429, 503, 201}, wantAttempts: 3},
		{name: "5xx until retries are exhausted", statuses: []int{500, 502, 503, 201}, wantAttempts: 3, wantStatus: 503, wantErr: true},
		{name: "400 is not retried", statuses: []int{400, 201}, wantAttempts: 1, wantStatus: 400, wantErr: true},
		{name: "401 is not retried", statuses: []int{401, 201}, wantAttempts: 1, wantStatus: 401, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attempts atomic.Int32

			c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				status := tt.statuses[attempts.Add(1)-1]
				writeJSON(w, status, ReimportResult{TestID: 1})
			})

			_, err := c.Reimport(context.Background(), testRequest())

			assert.Equal(t, tt.wantAttempts, attempts.Load())

			if !tt.wantErr {
				require.NoError(t, err)

				return
			}

			var apiErr *APIError

			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.StatusCode)
		})
	}
}

func TestClient_Reimport_RetriesTransportError(t *testing.T) {
	var attempts atomic.Int32

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			require.NoError(t, err)
			require.NoError(t, conn.Close())

			return
		}

		writeJSON(w, http.StatusCreated, ReimportResult{TestID: 1})
	})

	_, err := c.Reimport(context.Background(), testRequest())

	require.NoError(t, err)
	assert.Equal(t, int32(2), attempts.Load())
}

func TestClient_Reimport_StopsWaitingWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		cancel()
		w.WriteHeader(http.StatusInternalServerError)
	}, WithRetry(5, time.Hour))

	_, err := c.Reimport(ctx, testRequest())

	require.ErrorIs(t, err, context.Canceled)
}

func TestIsRejectedReport(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "bad request", err: &APIError{StatusCode: 400}, want: true},
		{name: "unprocessable", err: &APIError{StatusCode: 422}, want: true},
		{name: "unauthorized", err: &APIError{StatusCode: 401}},
		{name: "forbidden", err: &APIError{StatusCode: 403}},
		{name: "too many requests", err: &APIError{StatusCode: 429}},
		{name: "server error", err: &APIError{StatusCode: 500}},
		{name: "transport error", err: fmt.Errorf("connection refused")},
		{name: "wrapped", err: fmt.Errorf("wrap: %w", &APIError{StatusCode: 400}), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRejectedReport(tt.err))
		})
	}
}

// inFlightTracker counts concurrent requests per test title.
type inFlightTracker struct {
	mu      sync.Mutex
	current map[string]int
	maxSeen map[string]int
	total   int
	maxAll  int
	hold    time.Duration
}

func newInFlightTracker(hold time.Duration) *inFlightTracker {
	return &inFlightTracker{current: map[string]int{}, maxSeen: map[string]int{}, hold: hold}
}

func (f *inFlightTracker) handler(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("test_title")

	f.mu.Lock()
	f.current[title]++
	f.total++
	f.maxSeen[title] = max(f.maxSeen[title], f.current[title])
	f.maxAll = max(f.maxAll, f.total)
	f.mu.Unlock()

	time.Sleep(f.hold)

	f.mu.Lock()
	f.current[title]--
	f.total--
	f.mu.Unlock()

	writeJSON(w, http.StatusCreated, ReimportResult{TestID: 1})
}

func runConcurrently(t *testing.T, c *Client, titles []string) {
	t.Helper()

	var wg sync.WaitGroup

	for _, title := range titles {
		wg.Add(1)

		go func() {
			defer wg.Done()

			r := testRequest()
			r.TestTitle = title

			_, err := c.Reimport(context.Background(), r)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
}

func TestClient_Reimport_OneImportAtATimePerTest(t *testing.T) {
	tracker := newInFlightTracker(20 * time.Millisecond)
	c := newTestClient(t, tracker.handler)

	_, err := c.Reimport(context.Background(), testRequest())
	require.NoError(t, err)

	runConcurrently(t, c, []string{"same", "same", "same", "same"})

	assert.Equal(t, 1, tracker.maxSeen["same"])
}

func TestClient_Reimport_DifferentTestsOfKnownContextRunInParallel(t *testing.T) {
	tracker := newInFlightTracker(50 * time.Millisecond)
	c := newTestClient(t, tracker.handler)

	_, err := c.Reimport(context.Background(), testRequest())
	require.NoError(t, err)

	runConcurrently(t, c, []string{"a", "b", "c"})

	assert.Greater(t, tracker.maxAll, 1)
}

func TestClient_Reimport_FirstImportsOfAnUnknownContextAreSerialised(t *testing.T) {
	tracker := newInFlightTracker(30 * time.Millisecond)
	c := newTestClient(t, tracker.handler)

	runConcurrently(t, c, []string{"a", "b", "c"})

	assert.Equal(t, 1, tracker.maxAll)
}

func TestClient_Reimport_FailedFirstImportKeepsContextUnknown(t *testing.T) {
	var fail atomic.Bool

	fail.Store(true)

	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		writeJSON(w, http.StatusCreated, ReimportResult{TestID: 1})
	})

	_, err := c.Reimport(context.Background(), testRequest())
	require.Error(t, err)
	assert.False(t, c.isKnown("app\x00runtime-cdp-dev"))

	fail.Store(false)

	_, err = c.Reimport(context.Background(), testRequest())
	require.NoError(t, err)
	assert.True(t, c.isKnown("app\x00runtime-cdp-dev"))
}

// findingsServer serves the active findings of test 42 page by page and records the closed ones.
type findingsServer struct {
	t      *testing.T
	mu     sync.Mutex
	active []map[string]any
	closed []int
	bodies []map[string]any
	posts  atomic.Int32
	failOn int32
}

func (f *findingsServer) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	assert.Equal(f.t, "Token "+testToken, r.Header.Get("Authorization"))

	if r.Method == http.MethodGet {
		assert.Equal(f.t, "/api/v2/findings/", r.URL.Path)
		assert.Equal(f.t, "42", r.URL.Query().Get("test"))
		assert.Equal(f.t, "true", r.URL.Query().Get("active"))

		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		end := min(offset+limit, len(f.active))

		var next *string

		if end < len(f.active) {
			n := "http://next"
			next = &n
		}

		writeJSON(w, http.StatusOK, map[string]any{"next": next, "results": f.active[offset:end]})

		return
	}

	if f.posts.Add(1) == f.failOn {
		w.WriteHeader(http.StatusServiceUnavailable)

		return
	}

	var id int

	_, err := fmt.Sscanf(r.URL.Path, "/api/v2/findings/%d/close/", &id)
	require.NoError(f.t, err)

	body := map[string]any{}
	require.NoError(f.t, json.NewDecoder(r.Body).Decode(&body))

	f.bodies = append(f.bodies, body)
	f.closed = append(f.closed, id)
	f.active = slices.DeleteFunc(f.active, func(a map[string]any) bool { return a["id"] == float64(id) || a["id"] == id })

	writeJSON(w, http.StatusOK, map[string]any{})
}

func newFindingsServer(t *testing.T, services ...string) *findingsServer {
	t.Helper()

	f := &findingsServer{t: t}
	for i, s := range services {
		f.active = append(f.active, map[string]any{"id": i + 1, "service": s})
	}

	return f
}

func TestClient_CloseTest_ClosesEveryActiveFinding(t *testing.T) {
	services := make([]string, 250)
	f := newFindingsServer(t, services...)
	c := newTestClient(t, f.handle)

	require.NoError(t, c.CloseTest(context.Background(), 42))

	assert.Len(t, f.closed, 250, "findings of every page are closed")
	assert.Empty(t, f.active)
	require.Len(t, f.bodies, 250)

	for _, b := range f.bodies {
		assert.Equal(t, true, b["is_mitigated"])

		mitigated, ok := b["mitigated"].(string)
		require.True(t, ok)

		_, err := time.Parse(time.RFC3339, mitigated)
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(mitigated, "Z"))
	}
}

func TestClient_CloseTest_NoActiveFindings(t *testing.T) {
	f := newFindingsServer(t)
	c := newTestClient(t, f.handle)

	require.NoError(t, c.CloseTest(context.Background(), 42))
	assert.Zero(t, f.posts.Load())
}

func TestClient_CloseTest_RetriesClose(t *testing.T) {
	f := newFindingsServer(t, "a")
	f.failOn = 1
	c := newTestClient(t, f.handle)

	require.NoError(t, c.CloseTest(context.Background(), 42))

	assert.Equal(t, []int{1}, f.closed)
	assert.Equal(t, int32(2), f.posts.Load(), "the 503 was retried")
}

func TestClient_CloseTest_ServerError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	var apiErr *APIError

	require.ErrorAs(t, c.CloseTest(context.Background(), 42), &apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}

func TestClient_CloseFindingsOutsideService(t *testing.T) {
	f := newFindingsServer(t, "ns/Deployment/app/app", "ns/ReplicaSet/app-1/app", "ns/Deployment/app/app", "ns/ReplicaSet/app-2/app")
	c := newTestClient(t, f.handle)

	closed, err := c.CloseFindingsOutsideService(context.Background(), 42, "ns/Deployment/app/app")

	require.NoError(t, err)
	assert.Equal(t, 2, closed)
	assert.Equal(t, []int{2, 4}, f.closed)
}

func TestClient_CloseFindingsOutsideService_NothingToClose(t *testing.T) {
	f := newFindingsServer(t, "ns/Deployment/app/app")
	c := newTestClient(t, f.handle)

	closed, err := c.CloseFindingsOutsideService(context.Background(), 42, "ns/Deployment/app/app")

	require.NoError(t, err)
	assert.Zero(t, closed)
	assert.Zero(t, f.posts.Load())
}

func TestClient_ListTests_FollowsPages(t *testing.T) {
	var queries []string

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token "+testToken, r.Header.Get("Authorization"))
		assert.Equal(t, "krci-tenant:krci", r.URL.Query().Get("tags"))

		queries = append(queries, r.URL.RawQuery)

		offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
		require.NoError(t, err)

		if offset == 0 {
			writeJSON(w, http.StatusOK, map[string]any{
				"next":    "http://next",
				"results": []Test{{ID: 1, Title: "a", Tags: []string{"krci-env:cdp.dev"}}},
			})

			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"next":    nil,
			"results": []Test{{ID: 2, Title: "b"}},
		})
	})

	tests, err := c.ListTests(context.Background(), "krci-tenant:krci")

	require.NoError(t, err)
	assert.Equal(t, []Test{{ID: 1, Title: "a", Tags: []string{"krci-env:cdp.dev"}}, {ID: 2, Title: "b"}}, tests)
	assert.Len(t, queries, 2)
}

func TestClient_ListTests_ServerError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})

	_, err := c.ListTests(context.Background(), "krci-tenant:krci")

	var apiErr *APIError

	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadGateway, apiErr.StatusCode)
}

func TestTest_Tags(t *testing.T) {
	test := Test{Tags: []string{"krci-scope:runtime", "krci-env:cdp.dev"}}

	assert.True(t, test.HasTag("krci-scope:runtime"))
	assert.False(t, test.HasTag("krci-scope:ci"))

	v, ok := test.TagValue("krci-env")
	assert.True(t, ok)
	assert.Equal(t, "cdp.dev", v)

	_, ok = test.TagValue("krci-codebase")
	assert.False(t, ok)
}

func TestClient_GetTest(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v2/tests/42/", r.URL.Path)
		assert.Equal(t, "Token "+testToken, r.Header.Get("Authorization"))

		writeJSON(w, http.StatusOK, Test{ID: 42, Title: "a", Tags: []string{"krci-tenant:krci"}})
	})

	test, err := c.GetTest(context.Background(), 42)

	require.NoError(t, err)
	assert.Equal(t, Test{ID: 42, Title: "a", Tags: []string{"krci-tenant:krci"}}, test)
}

func TestClient_GetTest_NotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetTest(context.Background(), 42)

	var apiErr *APIError

	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}
