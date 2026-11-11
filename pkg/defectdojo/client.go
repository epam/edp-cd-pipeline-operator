// Package defectdojo is a minimal DefectDojo API v2 client for re-importing scan reports.
package defectdojo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// ScanTypeTrivyOperator is the DefectDojo parser for Trivy Operator reports.
	ScanTypeTrivyOperator = "Trivy Operator Scan"

	boolTrue = "true"

	reimportPath = "/api/v2/reimport-scan/"
	testsPath    = "/api/v2/tests/"
	findingsPath = "/api/v2/findings/"

	defaultRequestTimeout = 2 * time.Minute
	defaultMaxRetries     = 3
	defaultBackoff        = time.Second
	maxBackoff            = 30 * time.Second
	pageSize              = 100
)

// APIError is a non-2xx answer of DefectDojo.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("defectdojo answered %d: %s", e.StatusCode, e.Body)
}

// IsRejectedReport reports whether DefectDojo refused this one request (for example a malformed report)
// while the server itself is healthy, so the next request can still succeed.
func IsRejectedReport(err error) bool {
	var apiErr *APIError

	if !errors.As(err, &apiErr) {
		return false
	}

	switch apiErr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusRequestTimeout, http.StatusTooManyRequests:
		return false
	}

	return apiErr.StatusCode >= http.StatusBadRequest && apiErr.StatusCode < http.StatusInternalServerError
}

// ReimportRequest describes one re-import of a report into a DefectDojo test.
type ReimportRequest struct {
	ProductType string
	Product     string
	Engagement  string
	TestTitle   string
	Version     string
	Tags        []string
	ScanDate    time.Time
	// Report is the JSON document handed to the Trivy Operator parser.
	Report []byte
}

// ReimportResult holds the identifiers DefectDojo answers with.
type ReimportResult struct {
	TestID       int `json:"test_id"`
	EngagementID int `json:"engagement_id"`
	ProductID    int `json:"product_id"`
}

// Test is a DefectDojo test as returned by the tests list.
type Test struct {
	ID    int      `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

// HasTag reports whether the test carries the exact tag.
func (t Test) HasTag(tag string) bool {
	for _, have := range t.Tags {
		if have == tag {
			return true
		}
	}

	return false
}

// TagValue returns the value of the first tag with the given "<prefix>:" and whether it exists.
func (t Test) TagValue(prefix string) (string, bool) {
	for _, have := range t.Tags {
		if v, ok := strings.CutPrefix(have, prefix+":"); ok {
			return v, true
		}
	}

	return "", false
}

// state is shared by every Client derived with WithCredentials.
type state struct {
	httpClient *http.Client
	maxRetries int
	backoff    time.Duration
	sleep      func(ctx context.Context, d time.Duration) error

	// Imports into one test run one at a time: concurrent re-imports duplicate findings.
	testLocks keyedMutex
	// DefectDojo creates the product type, product and engagement of a first import
	// non-atomically, so first imports of unknown contexts run one at a time.
	createMu sync.Mutex
	mu       sync.Mutex
	known    map[string]struct{}
}

// Client talks to one DefectDojo instance.
type Client struct {
	baseURL string
	token   string
	*state
}

// Option configures a Client.
type Option func(*state)

// WithHTTPClient replaces the HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(s *state) { s.httpClient = c }
}

// WithRetry sets the number of retries after the first attempt and the first backoff.
// The backoff doubles on every retry.
func WithRetry(maxRetries int, backoff time.Duration) Option {
	return func(s *state) {
		s.maxRetries = maxRetries
		s.backoff = backoff
	}
}

// New returns a Client without credentials; use WithCredentials to bind it to an instance.
func New(opts ...Option) *Client {
	s := &state{
		httpClient: &http.Client{Timeout: defaultRequestTimeout},
		maxRetries: defaultMaxRetries,
		backoff:    defaultBackoff,
		sleep:      sleep,
		known:      map[string]struct{}{},
	}

	for _, o := range opts {
		o(s)
	}

	return &Client{state: s}
}

// WithCredentials returns a Client for the instance at baseURL that shares locks and state with c.
// Credentials are passed per call so a rotated Secret takes effect without a restart.
func (c *Client) WithCredentials(baseURL, token string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, state: c.state}
}

// Reimport re-imports the report into the test named by the request, creating the product type,
// product, engagement and test when they do not exist, and closes findings absent from the report.
func (c *Client) Reimport(ctx context.Context, r ReimportRequest) (ReimportResult, error) {
	fields := make([][2]string, 0, 15+len(r.Tags))
	fields = append(fields, [][2]string{
		{"scan_type", ScanTypeTrivyOperator},
		{"product_type_name", r.ProductType},
		{"product_name", r.Product},
		{"engagement_name", r.Engagement},
		{"test_title", r.TestTitle},
		{"version", r.Version},
		{"auto_create_context", boolTrue},
		{"close_old_findings", boolTrue},
		// Each environment is its own deduplication scope.
		{"deduplication_on_engagement", boolTrue},
		{"active", boolTrue},
		{"verified", "false"},
		{"minimum_severity", "Info"},
		{"environment", "Development"},
		{"push_to_jira", "false"},
		{"scan_date", r.ScanDate.Format(time.DateOnly)},
	}...)

	for _, tag := range r.Tags {
		fields = append(fields, [2]string{"tags", tag})
	}

	contextKey := r.Product + "\x00" + r.Engagement

	unlock := c.testLocks.lock(contextKey + "\x00" + r.TestTitle)
	defer unlock()

	if !c.isKnown(contextKey) {
		c.createMu.Lock()
		defer c.createMu.Unlock()
	}

	res, err := c.postReport(ctx, fields, r.Report)
	if err != nil {
		return ReimportResult{}, err
	}

	c.markKnown(contextKey)

	return res, nil
}

// CloseTest mitigates every active finding of the test. A test without active findings is closed.
func (c *Client) CloseTest(ctx context.Context, testID int) error {
	_, err := c.closeActiveFindings(ctx, testID, func(string) bool { return true })

	return err
}

// CloseFindingsOutsideService mitigates the active findings of the test whose service is not the given one
// and returns how many it closed.
func (c *Client) CloseFindingsOutsideService(ctx context.Context, testID int, service string) (int, error) {
	return c.closeActiveFindings(ctx, testID, func(s string) bool { return s != service })
}

func (c *Client) closeActiveFindings(ctx context.Context, testID int, selected func(service string) bool) (int, error) {
	unlock := c.testLocks.lock("test\x00" + strconv.Itoa(testID))
	defer unlock()

	// The list is read completely before anything is closed so closing does not shift the pages.
	var ids []int

	for offset := 0; ; offset += pageSize {
		q := url.Values{}
		q.Set("test", strconv.Itoa(testID))
		q.Set("active", "true")
		q.Set("limit", strconv.Itoa(pageSize))
		q.Set("offset", strconv.Itoa(offset))

		body, err := c.do(ctx, func() (*http.Request, error) {
			return c.newRequest(ctx, http.MethodGet, findingsPath+"?"+q.Encode(), nil, "")
		})
		if err != nil {
			return 0, err
		}

		var page struct {
			Next    *string `json:"next"`
			Results []struct {
				ID      int    `json:"id"`
				Service string `json:"service"`
			} `json:"results"`
		}

		if err = json.Unmarshal(body, &page); err != nil {
			return 0, fmt.Errorf("failed to decode findings list: %w", err)
		}

		for _, f := range page.Results {
			if selected(f.Service) {
				ids = append(ids, f.ID)
			}
		}

		if page.Next == nil || len(page.Results) == 0 {
			break
		}
	}

	for _, id := range ids {
		if err := c.closeFinding(ctx, id); err != nil {
			return 0, err
		}
	}

	return len(ids), nil
}

func (c *Client) closeFinding(ctx context.Context, findingID int) error {
	payload, err := json.Marshal(map[string]any{
		"is_mitigated": true,
		"mitigated":    time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("failed to encode close request: %w", err)
	}

	_, err = c.do(ctx, func() (*http.Request, error) {
		return c.newRequest(ctx, http.MethodPost, fmt.Sprintf("%s%d/close/", findingsPath, findingID),
			bytes.NewReader(payload), "application/json")
	})

	return err
}

// GetTest returns the test with the given ID.
func (c *Client) GetTest(ctx context.Context, testID int) (Test, error) {
	body, err := c.do(ctx, func() (*http.Request, error) {
		return c.newRequest(ctx, http.MethodGet, fmt.Sprintf("%s%d/", testsPath, testID), nil, "")
	})
	if err != nil {
		return Test{}, err
	}

	var test Test
	if err = json.Unmarshal(body, &test); err != nil {
		return Test{}, fmt.Errorf("failed to decode test %d: %w", testID, err)
	}

	return test, nil
}

// ListTests returns the tests that carry the tag. DefectDojo matches tags case-insensitively
// and by exact value; callers that need an exact match check Test.HasTag.
func (c *Client) ListTests(ctx context.Context, tag string) ([]Test, error) {
	var tests []Test

	for offset := 0; ; offset += pageSize {
		q := url.Values{}
		q.Set("tags", tag)
		q.Set("limit", strconv.Itoa(pageSize))
		q.Set("offset", strconv.Itoa(offset))

		body, err := c.do(ctx, func() (*http.Request, error) {
			return c.newRequest(ctx, http.MethodGet, testsPath+"?"+q.Encode(), nil, "")
		})
		if err != nil {
			return nil, err
		}

		var page struct {
			Next    *string `json:"next"`
			Results []Test  `json:"results"`
		}

		if err = json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("failed to decode tests list: %w", err)
		}

		tests = append(tests, page.Results...)

		if page.Next == nil || len(page.Results) == 0 {
			return tests, nil
		}
	}
}

func (c *Client) postReport(ctx context.Context, fields [][2]string, report []byte) (ReimportResult, error) {
	body, contentType, err := multipartBody(fields, report)
	if err != nil {
		return ReimportResult{}, err
	}

	resp, err := c.do(ctx, func() (*http.Request, error) {
		return c.newRequest(ctx, http.MethodPost, reimportPath, bytes.NewReader(body), contentType)
	})
	if err != nil {
		return ReimportResult{}, err
	}

	var res ReimportResult
	if err = json.Unmarshal(resp, &res); err != nil {
		return ReimportResult{}, fmt.Errorf("failed to decode re-import response: %w", err)
	}

	return res, nil
}

func (c *Client) newRequest(
	ctx context.Context,
	method, path string,
	body io.Reader,
	contentType string,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+c.token)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return req, nil
}

// do sends the request built by build, retrying 5xx, 429 and transport errors with exponential backoff.
func (c *Client) do(ctx context.Context, build func() (*http.Request, error)) ([]byte, error) {
	backoff := c.backoff

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := build()
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("request to defectdojo aborted: %w", ctx.Err())
			}

			lastErr = fmt.Errorf("request to defectdojo failed: %w", err)
		} else {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			switch {
			case readErr != nil:
				lastErr = fmt.Errorf("failed to read defectdojo response: %w", readErr)
			case resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices:
				return body, nil
			default:
				lastErr = &APIError{StatusCode: resp.StatusCode, Body: truncate(string(body))}
				if !retryable(resp.StatusCode) {
					return nil, lastErr
				}

				backoff = max(backoff, retryAfter(resp))
			}
		}

		if attempt == c.maxRetries {
			break
		}

		if err = c.sleep(ctx, min(backoff, maxBackoff)); err != nil {
			return nil, err
		}

		backoff *= 2
	}

	return nil, lastErr
}

func (c *Client) isKnown(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.known[key]

	return ok
}

func (c *Client) markKnown(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.known[key] = struct{}{}
}

func retryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func retryAfter(resp *http.Response) time.Duration {
	secs, err := strconv.Atoi(resp.Header.Get("Retry-After"))
	if err != nil || secs < 0 {
		return 0
	}

	return time.Duration(secs) * time.Second
}

func truncate(s string) string {
	const limit = 300

	if len(s) > limit {
		return s[:limit]
	}

	return s
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait for retry aborted: %w", ctx.Err())
	case <-t.C:
		return nil
	}
}

func multipartBody(fields [][2]string, report []byte) ([]byte, string, error) {
	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)

	for _, f := range fields {
		if err := w.WriteField(f[0], f[1]); err != nil {
			return nil, "", fmt.Errorf("failed to write field %s: %w", f[0], err)
		}
	}

	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="file"; filename="report.json"`)
	h.Set("Content-Type", "application/json")

	part, err := w.CreatePart(h)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create file part: %w", err)
	}

	if _, err = part.Write(report); err != nil {
		return nil, "", fmt.Errorf("failed to write file part: %w", err)
	}

	if err = w.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart body: %w", err)
	}

	return buf.Bytes(), w.FormDataContentType(), nil
}

// keyedMutex serialises work per key.
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*keyedLock
}

type keyedLock struct {
	mu   sync.Mutex
	refs int
}

func (k *keyedMutex) lock(key string) func() {
	k.mu.Lock()

	if k.locks == nil {
		k.locks = map[string]*keyedLock{}
	}

	l, ok := k.locks[key]
	if !ok {
		l = &keyedLock{}
		k.locks[key] = l
	}

	l.refs++
	k.mu.Unlock()

	l.mu.Lock()

	return func() {
		l.mu.Unlock()
		k.mu.Lock()

		l.refs--
		if l.refs == 0 {
			delete(k.locks, key)
		}

		k.mu.Unlock()
	}
}
