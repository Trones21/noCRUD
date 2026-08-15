// Package testutil provides a stand-in for a DRF backend, so the runner's own
// tests can exercise real HTTP without a Django app or a database.
//
// It is deliberately strict about the things noCRUD has to get right: it
// refuses unauthenticated writes, and it refuses writes without the CSRF header
// echoed back from the login cookie.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/dbclient"
)

// DRF is a fake Django REST Framework backend.
type DRF struct {
	Server *httptest.Server

	mu       sync.Mutex
	nextID   int
	objects  map[string]map[int]map[string]any
	requests []Request

	// Hooks let a test bend the behaviour for one endpoint.
	OnCreate func(resource string, obj map[string]any) (int, map[string]any, bool)
	OnUpdate func(resource string, id int, obj map[string]any) (map[string]any, bool)
}

// Request is one call the fake received.
type Request struct {
	Method string
	Path   string
	CSRF   string
	Body   map[string]any
}

const csrfToken = "test-csrf-token"

// NewDRF starts a fake backend and stops it when the test ends.
func NewDRF(t *testing.T) *DRF {
	t.Helper()

	d := &DRF{
		nextID:  1,
		objects: map[string]map[int]map[string]any{},
	}
	d.Server = httptest.NewServer(http.HandlerFunc(d.handle))
	t.Cleanup(d.Server.Close)
	return d
}

// Env returns an environment pointing flows at the fake backend.
func (d *DRF) Env() *nocrud.Env {
	u, err := url.Parse(d.Server.URL)
	if err != nil {
		panic(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		panic(err)
	}
	return &nocrud.Env{AppPort: port, DBName: "test", DBConfig: dbclient.Config{DBName: "test"}}
}

// BaseURL is the API root of the fake backend.
func (d *DRF) BaseURL() string { return d.Server.URL + "/api" }

// Requests returns everything the fake received, in order.
func (d *DRF) Requests() []Request {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Request(nil), d.requests...)
}

// Objects returns the stored objects for a resource.
func (d *DRF) Objects(resource string) map[int]map[string]any {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := map[int]map[string]any{}
	for id, obj := range d.objects[resource] {
		copied := map[string]any{}
		for k, v := range obj {
			copied[k] = v
		}
		out[id] = copied
	}
	return out
}

func (d *DRF) handle(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	d.mu.Lock()
	d.requests = append(d.requests, Request{
		Method: r.Method,
		Path:   r.URL.Path,
		CSRF:   r.Header.Get("X-CSRFToken"),
		Body:   body,
	})
	d.mu.Unlock()

	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/"), "/")

	if path == "login" {
		http.SetCookie(w, &http.Cookie{Name: "csrftoken", Value: csrfToken, Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "sessionid", Value: "test-session", Path: "/"})
		writeJSON(w, http.StatusOK, map[string]any{"detail": "logged in"})
		return
	}

	// Writes from a logged-in session must carry the CSRF token that login
	// handed out — the header wiring is the part most likely to break silently.
	//
	// Anonymous writes are not checked, matching DRF: SessionAuthentication
	// only enforces CSRF once a session actually authenticates a user, which is
	// what lets an unauthenticated client register a new one.
	if r.Method != http.MethodGet && hasSession(r) && r.Header.Get("X-CSRFToken") != csrfToken {
		writeJSON(w, http.StatusForbidden, map[string]any{"detail": "CSRF Failed: CSRF token missing or incorrect."})
		return
	}

	parts := strings.Split(path, "/")
	resource := parts[0]
	if idx := strings.Index(resource, "?"); idx >= 0 {
		resource = resource[:idx]
	}

	var id int
	hasID := false
	if len(parts) > 1 && parts[1] != "" {
		parsed, err := strconv.Atoi(parts[1])
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"detail": "Not found."})
			return
		}
		id, hasID = parsed, true
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	switch r.Method {
	case http.MethodPost:
		d.create(w, resource, body)
	case http.MethodGet:
		d.read(w, r, resource, id, hasID)
	case http.MethodPut, http.MethodPatch:
		d.update(w, resource, id, body)
	case http.MethodDelete:
		d.delete(w, resource, id, hasID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "Method not allowed."})
	}
}

func (d *DRF) create(w http.ResponseWriter, resource string, body map[string]any) {
	if d.OnCreate != nil {
		if status, res, handled := d.OnCreate(resource, body); handled {
			writeJSON(w, status, res)
			return
		}
	}

	obj := map[string]any{}
	for k, v := range body {
		obj[k] = v
	}
	id := d.nextID
	d.nextID++
	obj["id"] = float64(id)

	if d.objects[resource] == nil {
		d.objects[resource] = map[int]map[string]any{}
	}
	d.objects[resource][id] = obj
	writeJSON(w, http.StatusCreated, obj)
}

func (d *DRF) read(w http.ResponseWriter, r *http.Request, resource string, id int, hasID bool) {
	if hasID {
		obj, ok := d.objects[resource][id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"detail": "Not found."})
			return
		}
		writeJSON(w, http.StatusOK, obj)
		return
	}

	// List, DRF-paginated. Honours ?field=value so lookups can be tested.
	var results []any
	for _, obj := range d.objects[resource] {
		if matchesQuery(obj, r.URL.Query()) {
			results = append(results, obj)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":   len(results),
		"results": results,
	})
}

func (d *DRF) update(w http.ResponseWriter, resource string, id int, body map[string]any) {
	obj, ok := d.objects[resource][id]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "Not found."})
		return
	}
	if d.OnUpdate != nil {
		if res, handled := d.OnUpdate(resource, id, body); handled {
			writeJSON(w, http.StatusOK, res)
			return
		}
	}
	for k, v := range body {
		obj[k] = v
	}
	obj["id"] = float64(id)
	writeJSON(w, http.StatusOK, obj)
}

func (d *DRF) delete(w http.ResponseWriter, resource string, id int, hasID bool) {
	if !hasID {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "A pk is required."})
		return
	}
	if _, ok := d.objects[resource][id]; !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "Not found."})
		return
	}
	delete(d.objects[resource], id)
	w.WriteHeader(http.StatusNoContent)
}

func hasSession(r *http.Request) bool {
	_, err := r.Cookie("sessionid")
	return err == nil
}

func matchesQuery(obj map[string]any, query url.Values) bool {
	for key, want := range query {
		got, ok := obj[key]
		if !ok || len(want) == 0 {
			return false
		}
		if toString(got) != want[0] {
			return false
		}
	}
	return true
}

func toString(v any) string {
	switch n := v.(type) {
	case string:
		return n
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	default:
		return ""
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
