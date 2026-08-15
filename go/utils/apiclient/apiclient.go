// Package apiclient talks to the backend the way your frontend does — the Go
// counterpart of python/utils/api_client.py.
//
// One Client is one logged-in user, holding its own cookie jar and CSRF token.
// Multi-user flows are just several Clients: create one per user, and the
// requests interleave however the flow says they should.
//
// There aren't many authentication options here yet — the ones the example app
// needed. That's the point of shipping source rather than a library: open this
// file and make Login do whatever your backend expects.
package apiclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Trones21/noCRUD/go/utils/jsonx"
	"github.com/Trones21/noCRUD/go/utils/misc"
	"github.com/Trones21/noCRUD/go/utils/perf"
)

// Options configure a Client.
//
// In parallel mode every flow has its own backend on its own port and its own
// output buffer, so all three of these are per-flow values. The Python runner
// reads them from process globals (os.environ, stdout); goroutines share those,
// so here they are passed explicitly.
type Options struct {
	// BaseURL is the API root, e.g. http://localhost:8000/api. Empty means
	// http://localhost:$APP_PORT/api, defaulting to port 8000.
	BaseURL string
	// Out is where request logging goes. Empty means os.Stdout.
	Out io.Writer
	// Perf collects timings for the flow this client belongs to. Nil disables
	// collection.
	Perf *perf.Collector
	// Timeout caps every request. Zero means 30s.
	Timeout time.Duration
	// CSRFCookie / CSRFHeader name the CSRF pair. Empty means Django's
	// defaults: the csrftoken cookie, echoed back as X-CSRFToken.
	CSRFCookie string
	CSRFHeader string
}

// Client is one API session.
type Client struct {
	BaseURL  string
	Username string

	http       *http.Client
	out        io.Writer
	perf       *perf.Collector
	csrfCookie string
	csrfHeader string
	headers    http.Header
}

// New builds a client with its own cookie jar.
func New(opts Options) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimSuffix(opts.BaseURL, "/")
	if baseURL == "" {
		port := os.Getenv("APP_PORT")
		if port == "" {
			port = "8000"
		}
		baseURL = fmt.Sprintf("http://localhost:%s/api", port)
	}
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	csrfCookie := opts.CSRFCookie
	if csrfCookie == "" {
		csrfCookie = "csrftoken"
	}
	csrfHeader := opts.CSRFHeader
	if csrfHeader == "" {
		csrfHeader = "X-CSRFToken"
	}

	return &Client{
		BaseURL:    baseURL,
		http:       &http.Client{Jar: jar, Timeout: timeout},
		out:        out,
		perf:       opts.Perf,
		csrfCookie: csrfCookie,
		csrfHeader: csrfHeader,
		headers:    http.Header{},
	}, nil
}

// SetHeader adds a header sent on every subsequent request — the hook for
// bearer tokens, API keys and the like.
func (c *Client) SetHeader(key, value string) { c.headers.Set(key, value) }

// Out returns the writer this client logs to, so callers can print into the
// same buffer.
func (c *Client) Out() io.Writer { return c.out }

// Login authenticates and captures the session cookie and CSRF token.
func (c *Client) Login(username, password string) error {
	c.Username = username
	res, _, err := c.do("POST", c.BaseURL+"/login/", map[string]any{
		"username": username,
		"password": password,
	})
	if err != nil {
		return err
	}
	_ = res

	// Echo the CSRF cookie back as a header on subsequent writes, the way a
	// browser does. The cookie itself is already in the jar.
	if token := c.cookie(c.csrfCookie); token != "" {
		c.headers.Set(c.csrfHeader, token)
	}

	fmt.Fprintf(c.out, "\nAPI Client Created With %s\n", username)
	return nil
}

// ============================================================================
//  Django standard DRF CRUD format
// ============================================================================

// CreateObject creates an object at the endpoint and returns the response.
func (c *Client) CreateObject(endpoint string, data any) (jsonx.Value, error) {
	res, elapsed, err := c.do("POST", fmt.Sprintf("%s/%s/", c.BaseURL, endpoint), data)
	c.record("create_object", "Create:", endpoint, elapsed)
	if err != nil {
		return res, err
	}
	fmt.Fprintf(c.out, "Object created: %v\n", res)
	return res, nil
}

// GetObjectByID reads an object by id, or lists the endpoint when id is empty.
func (c *Client) GetObjectByID(endpoint, id string, silent bool) (jsonx.Value, error) {
	url := fmt.Sprintf("%s/%s/", c.BaseURL, endpoint)
	if id != "" {
		url += id + "/"
	}
	res, elapsed, err := c.do("GET", url, nil)
	c.record("get_object_by_id", "Get:", endpoint, elapsed)
	if err != nil {
		return res, err
	}
	if !silent {
		fmt.Fprintf(c.out, "Objects retrieved: %v\n", res)
	}
	return res, nil
}

// UpdateObjectByID replaces an object by id (PUT).
func (c *Client) UpdateObjectByID(endpoint, id string, data any) (jsonx.Value, error) {
	res, elapsed, err := c.do("PUT", fmt.Sprintf("%s/%s/%s/", c.BaseURL, endpoint, id), data)
	c.record("update_object_by_id", "Update:", endpoint, elapsed)
	if err != nil {
		return res, err
	}
	fmt.Fprintf(c.out, "Object updated: %v\n", res)
	return res, nil
}

// PatchObjectByID partially updates an object by id.
func (c *Client) PatchObjectByID(endpoint, id string, data any) (jsonx.Value, error) {
	res, elapsed, err := c.do("PATCH", fmt.Sprintf("%s/%s/%s/", c.BaseURL, endpoint, id), data)
	c.record("patch_object_by_id", "Patch:", endpoint, elapsed)
	if err != nil {
		return res, err
	}
	fmt.Fprintf(c.out, "Object patched: %v\n", res)
	return res, nil
}

// DeleteObjectByID deletes an object by id.
func (c *Client) DeleteObjectByID(endpoint, id string) error {
	url := fmt.Sprintf("%s/%s/", c.BaseURL, endpoint)
	if id != "" {
		url += id + "/"
	}
	_, elapsed, err := c.do("DELETE", url, nil)
	c.record("delete_object_by_id", "Delete:", endpoint, elapsed)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "Object with ID %s deleted successfully.\n", id)
	return nil
}

// ============================================================================
//  Freeform endpoints — for URLs that don't follow DRF (common for "actions")
// ============================================================================

// Get sends a GET to an arbitrary path below the API root.
func (c *Client) Get(endpoint string, silent bool) (jsonx.Value, error) {
	res, elapsed, err := c.do("GET", fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
	c.record("get", "GET:", endpoint, elapsed)
	if err != nil {
		return res, err
	}
	if !silent {
		fmt.Fprintf(c.out, "%s: object/s retrieved: %v\n", c.Username, res)
	}
	return res, nil
}

// Post sends a POST to an arbitrary path below the API root.
func (c *Client) Post(endpoint string, data any, silent bool) (jsonx.Value, error) {
	res, elapsed, err := c.do("POST", fmt.Sprintf("%s/%s/", c.BaseURL, endpoint), data)
	c.record("post", "POST:", endpoint, elapsed)
	if err != nil {
		return res, err
	}
	if !silent {
		fmt.Fprintf(c.out, "%s: object/s created (POST): %v\n", c.Username, res)
	}
	return res, nil
}

// Put sends a PUT to an arbitrary path below the API root.
func (c *Client) Put(endpoint string, data any, silent bool) (jsonx.Value, error) {
	res, elapsed, err := c.do("PUT", fmt.Sprintf("%s/%s/", c.BaseURL, endpoint), data)
	c.record("put", "PUT:", endpoint, elapsed)
	if err != nil {
		return res, err
	}
	if !silent {
		fmt.Fprintf(c.out, "%s: object/s updated (PUT): %v\n", c.Username, res)
	}
	return res, nil
}

// Delete sends a DELETE to an arbitrary path below the API root.
func (c *Client) Delete(endpoint string, silent bool) error {
	res, elapsed, err := c.do("DELETE", fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
	c.record("delete", "DELETE:", endpoint, elapsed)
	if err != nil {
		return err
	}
	if !silent {
		fmt.Fprintf(c.out, "%s: object/s deleted: %v\n", c.Username, res)
	}
	return nil
}

// CleanupDelete deletes without treating a failure as an error.
//
// You might run this just to be sure a table is empty before writing to it —
// which naturally fails when it is already empty, and that failure means
// nothing, so it isn't surfaced.
func (c *Client) CleanupDelete(endpoint string) {
	_, _, _ = c.do("DELETE", fmt.Sprintf("%s/%s", c.BaseURL, endpoint), nil)
}

// ============================================================================
//  Convenience lookups
// ============================================================================

// GetUserIDViaUsername looks a user up by username, which is unique in the
// example backend, so there is no multiple-match case to handle.
func (c *Client) GetUserIDViaUsername(username string) (string, error) {
	user, err := c.GetUserViaUsername(username)
	if err != nil {
		return "", err
	}
	return user.Get("id").ID(), nil
}

// GetUserViaUsername looks a user up by username.
func (c *Client) GetUserViaUsername(username string) (jsonx.Value, error) {
	res, err := c.Get("users?username="+url.QueryEscape(username), true)
	if err != nil {
		return res, err
	}
	user := res.Get("results.0")
	if err := user.Err(); err != nil {
		return user, fmt.Errorf("no user found with username %q", username)
	}
	return user, nil
}

// ============================================================================
//  Constructors that also bring a user with them
// ============================================================================

// NewWithUserViaFixture creates a Client logged in as a user from a fixture.
//
// The user must already exist — load the fixtures first (Ctx.Setup does).
func NewWithUserViaFixture(opts Options, username, password string) (*Client, error) {
	c, err := New(opts)
	if err != nil {
		return nil, err
	}
	if err := c.Login(username, password); err != nil {
		return nil, err
	}
	return c, nil
}

// NewWithNewRandomUser creates a Client backed by a freshly registered random
// user, which is how a multi-user flow gets its second and third actors without
// needing more fixtures.
func NewWithNewRandomUser(opts Options) (*Client, error) {
	c, err := New(opts)
	if err != nil {
		return nil, err
	}

	password := misc.RandomString(12)
	username := misc.RandomString(12)
	res, err := c.CreateObject("users", map[string]any{
		"username": username,
		"email":    misc.RandomString(12) + "@example.com",
		"password": password,
	})
	if err != nil {
		return nil, err
	}
	if created := res.Get("username").Text(); created != "" {
		username = created
	}
	if err := c.Login(username, password); err != nil {
		return nil, err
	}
	return c, nil
}

// ============================================================================
//  Plumbing
// ============================================================================

// do performs one request: encode, send, decode, and report a non-2xx as an
// *HTTPError with the body attached.
func (c *Client) do(method, url string, body any) (jsonx.Value, time.Duration, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return jsonx.Invalid(err), 0, fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return jsonx.Invalid(err), 0, err
	}
	for k, v := range c.headers {
		req.Header[k] = v
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	res, err := c.http.Do(req)
	if err != nil {
		return jsonx.Invalid(err), time.Since(start), &ConnectionError{Method: method, URL: url, Err: err}
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	elapsed := time.Since(start)
	if err != nil {
		return jsonx.Invalid(err), elapsed, err
	}

	// Refresh the CSRF header whenever the backend rotates the cookie.
	if token := c.cookie(c.csrfCookie); token != "" {
		c.headers.Set(c.csrfHeader, token)
	}

	if res.StatusCode > 299 {
		httpErr := &HTTPError{
			Method:     method,
			URL:        url,
			StatusCode: res.StatusCode,
			Status:     res.Status,
			Body:       string(raw),
		}
		c.printOnFailStatus(httpErr)
		return jsonx.Invalid(httpErr), elapsed, httpErr
	}

	// 204 No Content and friends have nothing to decode.
	if len(bytes.TrimSpace(raw)) == 0 {
		return jsonx.New(nil), elapsed, nil
	}
	return jsonx.Unmarshal(raw), elapsed, nil
}

// record prints the inline timing and persists it for regression tracking.
// Persisting is best-effort: perf must never break a run.
func (c *Client) record(op, label, endpoint string, elapsed time.Duration) {
	fmt.Fprintf(c.out, "%s %.2f ms\n\n", label, float64(elapsed.Nanoseconds())/1e6)
	c.perf.Record(op, endpoint, elapsed)
}

// cookie reads a cookie by name out of the jar.
func (c *Client) cookie(name string) string {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return ""
	}
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

// printOnFailStatus logs the exact response, pretty-printing the body when it
// is JSON — which for DRF is the field-level validation errors you actually
// want to read.
func (c *Client) printOnFailStatus(e *HTTPError) {
	fmt.Fprintf(c.out, "🔴 ERROR %d: %s\n", e.StatusCode, e.Body)
	var parsed any
	if err := json.Unmarshal([]byte(e.Body), &parsed); err != nil {
		fmt.Fprintf(c.out, "⚠️ Response is not JSON: %s\n", e.Body)
		return
	}
	fmt.Fprintf(c.out, "🔍 Error Details: %s\n", jsonx.New(parsed).Pretty())
}

// ============================================================================
//  Errors
// ============================================================================

// HTTPError is a non-2xx response. The message matches the Python runner's
// (requests' raise_for_status), so failure summaries read the same in both.
type HTTPError struct {
	Method     string
	URL        string
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	kind := "Server Error"
	if e.StatusCode < 500 {
		kind = "Client Error"
	}
	reason := strings.TrimSpace(strings.TrimPrefix(e.Status, fmt.Sprint(e.StatusCode)))
	if reason == "" {
		reason = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("%d %s: %s for url: %s", e.StatusCode, kind, reason, e.URL)
}

// ConnectionError is a request that never got a response.
type ConnectionError struct {
	Method string
	URL    string
	Err    error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Method, e.URL, e.Err)
}

func (e *ConnectionError) Unwrap() error { return e.Err }

// IsConnectionError reports whether err means the backend could not be reached
// — the case worth telling the user "is it running?" about, rather than dumping
// a stack trace on.
func IsConnectionError(err error) bool {
	var connErr *ConnectionError
	if errors.As(err, &connErr) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// StatusOf returns the HTTP status carried by err, or 0 if it isn't an
// *HTTPError. Useful when a flow asserts a *particular* failure.
func StatusOf(err error) int {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode
	}
	return 0
}
