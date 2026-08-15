package fixtures_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Trones21/noCRUD/go/config"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
)

// A Django user fixture: the password is stored hashed, with the plaintext
// carried alongside it so the runner can actually log in.
const usersFixture = `[
  {
    "model": "api.users",
    "pk": 1,
    "unhashed_pass": "fixture_pass",
    "fields": {
      "username": "var_undecided",
      "first_name": "var",
      "password": "pbkdf2_sha256$260000$abc"
    }
  },
  {
    "model": "api.users",
    "pk": 2,
    "fields": {"username": "no_password_here"}
  }
]`

func withFixtures(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "users.json"), []byte(usersFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvFixturesPath, dir)
	return dir
}

func TestGetByIndexFoldsInTheUnhashedPassword(t *testing.T) {
	withFixtures(t)

	fields, err := fixtures.GetByIndex("users.json", 0)
	if err != nil {
		t.Fatalf("GetByIndex: %v", err)
	}
	if fields["username"] != "var_undecided" {
		t.Errorf("username = %v", fields["username"])
	}
	if fields["unhashed_pass"] != "fixture_pass" {
		t.Errorf("unhashed_pass = %v, want it folded into the fields", fields["unhashed_pass"])
	}
}

// Fixtures are read fresh each time: a flow that tweaks the object it got back
// must not affect the next flow to ask for it.
func TestGetByIndexReturnsAFreshCopy(t *testing.T) {
	withFixtures(t)

	first, err := fixtures.GetByIndex("users.json", 0)
	if err != nil {
		t.Fatal(err)
	}
	first["username"] = "mutated"

	second, err := fixtures.GetByIndex("users.json", 0)
	if err != nil {
		t.Fatal(err)
	}
	if second["username"] != "var_undecided" {
		t.Errorf("username = %v, want the mutation not to leak", second["username"])
	}
}

func TestCredentials(t *testing.T) {
	withFixtures(t)

	username, password, err := fixtures.Credentials("users.json", 0)
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if username != "var_undecided" || password != "fixture_pass" {
		t.Errorf("got %q/%q", username, password)
	}

	// The common Django trap: a fixture with only the hashed password can't be
	// logged in as, and the error should say why.
	_, _, err = fixtures.Credentials("users.json", 1)
	if err == nil {
		t.Fatal("expected an error for a fixture with no plaintext password")
	}
	if !strings.Contains(err.Error(), "unhashed_pass") {
		t.Errorf("error = %q, want it to explain the convention", err)
	}
}

func TestErrorsPointAtTheFile(t *testing.T) {
	dir := withFixtures(t)

	if _, err := fixtures.Get("nope.json"); err == nil || !strings.Contains(err.Error(), dir) {
		t.Errorf("error = %v, want it to name the path we looked in", err)
	}
	if _, err := fixtures.GetByIndex("users.json", 99); err == nil {
		t.Error("expected an out-of-range index to be reported")
	}

	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fixtures.Get("broken.json"); err == nil {
		t.Error("expected malformed JSON to be reported")
	}
}
