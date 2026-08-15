package provisioning_test

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Trones21/noCRUD/go/utils/provisioning"
)

func TestFindOpenPortReturnsUsablePorts(t *testing.T) {
	seen := map[int]bool{}
	for i := 0; i < 5; i++ {
		port, err := provisioning.FindOpenPort()
		if err != nil {
			t.Fatalf("FindOpenPort: %v", err)
		}
		if port <= 0 || port > 65535 {
			t.Fatalf("port = %d", port)
		}
		seen[port] = true

		// The port has to actually be bindable — that's the whole contract.
		l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			t.Fatalf("port %d was not free: %v", port, err)
		}
		l.Close()
	}
	if len(seen) < 2 {
		t.Errorf("got the same port every time: %v", seen)
	}
}

func TestEnsureMigrationsExist(t *testing.T) {
	appDir := t.TempDir()

	// An app whose migrations folder holds only __init__.py hasn't had
	// makemigrations run — migrating it would silently create no tables.
	empty := filepath.Join(appDir, "api", "migrations")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(empty, "__init__.py"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	err := provisioning.EnsureMigrationsExist(appDir)
	if err == nil {
		t.Fatal("expected an app with no migrations to be reported")
	}
	if !strings.Contains(err.Error(), "api") {
		t.Errorf("error = %q, want it to name the app", err)
	}

	if err := os.WriteFile(filepath.Join(empty, "0001_initial.py"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := provisioning.EnsureMigrationsExist(appDir); err != nil {
		t.Errorf("EnsureMigrationsExist = %v, want nil once a migration exists", err)
	}
}

// Directories that aren't apps (no migrations folder) are none of our business.
func TestEnsureMigrationsExistIgnoresNonApps(t *testing.T) {
	appDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(appDir, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := provisioning.EnsureMigrationsExist(appDir); err != nil {
		t.Errorf("EnsureMigrationsExist = %v, want nil", err)
	}
}

func TestEnsureSchemaDefinitionExistsExplainsHowToMakeOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.sql")

	err := provisioning.EnsureSchemaDefinitionExists(path)
	if err == nil {
		t.Fatal("expected a missing schema to be reported")
	}
	if !strings.Contains(err.Error(), "pg_dump") {
		t.Errorf("error = %q, want it to say how to generate the file", err)
	}

	if err := os.WriteFile(path, []byte("-- schema"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := provisioning.EnsureSchemaDefinitionExists(path); err != nil {
		t.Errorf("EnsureSchemaDefinitionExists = %v, want nil", err)
	}
}

func TestWaitForBackendToListen(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port
	if err := provisioning.WaitForBackendToListen(port, time.Second); err != nil {
		t.Errorf("WaitForBackendToListen on an open port = %v", err)
	}

	closed, err := provisioning.FindOpenPort()
	if err != nil {
		t.Fatal(err)
	}
	err = provisioning.WaitForBackendToListen(closed, 300*time.Millisecond)
	if err == nil {
		t.Fatal("expected a timeout on a port nothing is listening on")
	}
	if !strings.Contains(err.Error(), strconv.Itoa(closed)) {
		t.Errorf("error = %q, want it to name the port", err)
	}
}
