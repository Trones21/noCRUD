package apiclient_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Trones21/noCRUD/go/internal/testutil"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
)

func newClient(t *testing.T, drf *testutil.DRF) (*apiclient.Client, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	client, err := apiclient.New(apiclient.Options{BaseURL: drf.BaseURL(), Out: &out})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client, &out
}

func TestLoginCapturesCSRFTokenForLaterWrites(t *testing.T) {
	drf := testutil.NewDRF(t)
	client, _ := newClient(t, drf)

	if err := client.Login("var_undecided", "fixture_pass"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	// The fake rejects any write that doesn't echo the login cookie back as a
	// header, so a successful create proves the CSRF wiring works.
	if _, err := client.CreateObject("universe", map[string]any{"name": "Breaking Bad"}); err != nil {
		t.Fatalf("CreateObject after login: %v", err)
	}

	for _, req := range drf.Requests() {
		if req.Method == "POST" && strings.Contains(req.Path, "universe") && req.CSRF == "" {
			t.Error("create request went out without the X-CSRFToken header")
		}
	}
}

func TestPermissionDeniedIsSurfacedWithItsStatus(t *testing.T) {
	drf := testutil.NewDRF(t)
	drf.OnCreate = func(resource string, obj map[string]any) (int, map[string]any, bool) {
		return 403, map[string]any{"detail": "You do not have permission to perform this action."}, true
	}

	client, _ := newClient(t, drf)
	if err := client.Login("user", "pass"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	// A multi-user flow leans on this: "user B may not do X" is only an
	// assertion if the status comes back intact.
	_, err := client.CreateObject("universe", map[string]any{"name": "nope"})
	if err == nil {
		t.Fatal("expected the create to be refused")
	}
	if got := apiclient.StatusOf(err); got != 403 {
		t.Errorf("status = %d, want 403", got)
	}
}

func TestCRUDRoundTrip(t *testing.T) {
	drf := testutil.NewDRF(t)
	client, _ := newClient(t, drf)
	if err := client.Login("user", "pass"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	created, err := client.CreateObject("universe", map[string]any{"name": "Breaking Bad"})
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	id := created.Get("id").ID()
	if id != "1" {
		t.Fatalf("id = %q, want 1", id)
	}

	read, err := client.GetObjectByID("universe", id, true)
	if err != nil {
		t.Fatalf("GetObjectByID: %v", err)
	}
	if got := read.Get("name").Text(); got != "Breaking Bad" {
		t.Errorf("name = %q, want Breaking Bad", got)
	}

	updated, err := client.UpdateObjectByID("universe", id, map[string]any{"name": "A whole new world"})
	if err != nil {
		t.Fatalf("UpdateObjectByID: %v", err)
	}
	if got := updated.Get("name").Text(); got != "A whole new world" {
		t.Errorf("updated name = %q", got)
	}

	if err := client.DeleteObjectByID("universe", id); err != nil {
		t.Fatalf("DeleteObjectByID: %v", err)
	}
	if left := drf.Objects("universe"); len(left) != 0 {
		t.Errorf("object still present after delete: %v", left)
	}
}

// A 204 has no body to decode; the client must not treat that as a failure.
func TestDeleteHandlesEmptyResponseBody(t *testing.T) {
	drf := testutil.NewDRF(t)
	client, _ := newClient(t, drf)
	if err := client.Login("user", "pass"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if _, err := client.CreateObject("actor", map[string]any{"first_name": "Bryan"}); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	if err := client.DeleteObjectByID("actor", "1"); err != nil {
		t.Fatalf("DeleteObjectByID: %v", err)
	}
}

func TestErrorBodyIsPrintedAndCarried(t *testing.T) {
	drf := testutil.NewDRF(t)
	drf.OnCreate = func(resource string, obj map[string]any) (int, map[string]any, bool) {
		return 400, map[string]any{"production": []any{`Invalid pk "1" - object does not exist.`}}, true
	}

	client, out := newClient(t, drf)
	if err := client.Login("user", "pass"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	_, err := client.CreateObject("character", map[string]any{"name": "Walter"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := apiclient.StatusOf(err); got != 400 {
		t.Errorf("status = %d, want 400", got)
	}
	// The message shape matches the Python runner's, so summaries read the same.
	if !strings.Contains(err.Error(), "400 Client Error") {
		t.Errorf("error = %q, want it to mention 400 Client Error", err)
	}
	logged := out.String()
	if !strings.Contains(logged, "🔴 ERROR 400") || !strings.Contains(logged, "Invalid pk") {
		t.Errorf("failure log did not include the response body:\n%s", logged)
	}
}

func TestConnectionErrorIsRecognised(t *testing.T) {
	client, err := apiclient.New(apiclient.Options{BaseURL: "http://127.0.0.1:1/api", Out: &bytes.Buffer{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.GetObjectByID("universe", "1", true)
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if !apiclient.IsConnectionError(err) {
		t.Errorf("IsConnectionError(%v) = false, want true", err)
	}
}

func TestGetUserViaUsername(t *testing.T) {
	drf := testutil.NewDRF(t)
	client, _ := newClient(t, drf)
	if err := client.Login("user", "pass"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if _, err := client.CreateObject("users", map[string]any{"username": "var_undecided"}); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	id, err := client.GetUserIDViaUsername("var_undecided")
	if err != nil {
		t.Fatalf("GetUserIDViaUsername: %v", err)
	}
	if id != "1" {
		t.Errorf("id = %q, want 1", id)
	}

	if _, err := client.GetUserViaUsername("nobody"); err == nil {
		t.Error("expected an error looking up a user that doesn't exist")
	}
}

func TestNewWithNewRandomUserRegistersAndLogsIn(t *testing.T) {
	drf := testutil.NewDRF(t)
	var out bytes.Buffer

	client, err := apiclient.NewWithNewRandomUser(apiclient.Options{BaseURL: drf.BaseURL(), Out: &out})
	if err != nil {
		t.Fatalf("NewWithNewRandomUser: %v", err)
	}
	if client.Username == "" {
		t.Error("client has no username")
	}
	if users := drf.Objects("users"); len(users) != 1 {
		t.Errorf("created %d users, want 1", len(users))
	}
}
