package dbclient_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/Trones21/noCRUD/go/utils/dbclient"
	"github.com/Trones21/noCRUD/go/utils/misc"
)

// requirePostgres skips when there's no database to talk to, so the rest of the
// suite still runs on a machine (or a CI job) without one.
func requirePostgres(t *testing.T) *dbclient.Client {
	t.Helper()

	ctx := context.Background()
	cfg := dbclient.AdminConfig()
	if err := dbclient.WaitReady(ctx, cfg, 2*time.Second); err != nil {
		t.Skipf("no postgres available (%v)", err)
	}

	client, err := dbclient.New(ctx, cfg, io.Discard)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { client.Close(ctx) })
	return client
}

func TestConfigURL(t *testing.T) {
	cfg := dbclient.Config{DBName: "example", User: "postgres", Pass: "secret", Host: "localhost", Port: "5432"}
	if got, want := cfg.URL(), "postgres://postgres:secret@localhost:5432/example"; got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestAdminConfigTargetsTheMaintenanceDatabase(t *testing.T) {
	t.Setenv("DB_NAME", "noCRUD_p8001_actor")

	// Creating and dropping per-flow databases can't be done from inside one of
	// them, so the admin connection must ignore DB_NAME.
	if got := dbclient.AdminConfig().DBName; got != "postgres" {
		t.Errorf("admin DBName = %q, want postgres", got)
	}
	if got := dbclient.ConfigFromEnv().DBName; got != "noCRUD_p8001_actor" {
		t.Errorf("ConfigFromEnv DBName = %q", got)
	}
}

// The Go counterpart of python/utils/db_client_test.py.
func TestCreateAndDropLifecycle(t *testing.T) {
	client := requirePostgres(t)
	ctx := context.Background()
	name := "nocrud_testdb_" + misc.RandomString(6)

	if err := client.DropDB(ctx, name); err != nil { // in case a crash left one behind
		t.Fatalf("DropDB (pre-clean): %v", err)
	}
	if err := client.CreateDB(ctx, name); err != nil {
		t.Fatalf("CreateDB: %v", err)
	}
	t.Cleanup(func() { client.DropDB(ctx, name) })

	// Creating it twice is a warning, not a failure — two flows racing to
	// provision the same name shouldn't take the run down.
	if err := client.CreateDB(ctx, name); err != nil {
		t.Errorf("CreateDB on an existing database = %v, want nil", err)
	}
	if err := client.DropDB(ctx, name); err != nil {
		t.Errorf("DropDB: %v", err)
	}
	// And dropping a database that isn't there is fine too.
	if err := client.DropDB(ctx, name); err != nil {
		t.Errorf("DropDB on a missing database = %v, want nil", err)
	}
}

func TestClearAndDropTables(t *testing.T) {
	admin := requirePostgres(t)
	ctx := context.Background()
	name := "nocrud_testdb_" + misc.RandomString(6)

	if err := admin.CreateDB(ctx, name); err != nil {
		t.Fatalf("CreateDB: %v", err)
	}
	t.Cleanup(func() { admin.DropDB(ctx, name) })

	cfg := dbclient.AdminConfig()
	cfg.DBName = name
	client, err := dbclient.New(ctx, cfg, io.Discard)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close(ctx)

	seed := []string{
		`CREATE TABLE universe (id serial PRIMARY KEY, name text)`,
		`CREATE TABLE production (id serial PRIMARY KEY, universe_id int REFERENCES universe(id))`,
		`CREATE TABLE django_migrations (id serial PRIMARY KEY, app text)`,
		`INSERT INTO universe (name) VALUES ('Breaking Bad')`,
		`INSERT INTO production (universe_id) VALUES (1)`,
		`INSERT INTO django_migrations (app) VALUES ('api')`,
	}
	for _, stmt := range seed {
		if err := client.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}

	if err := client.ClearAllTables(ctx); err != nil {
		t.Fatalf("ClearAllTables: %v", err)
	}
	if err := client.VerifyAllTablesCleared(ctx); err != nil {
		t.Errorf("VerifyAllTablesCleared: %v", err)
	}

	// Django's own tables are left alone, or the app would need re-migrating
	// after every reset.
	var migrations int
	if err := client.QueryRow(ctx, `SELECT COUNT(*) FROM django_migrations`).Scan(&migrations); err != nil {
		t.Fatalf("counting django_migrations: %v", err)
	}
	if migrations != 1 {
		t.Errorf("django_migrations has %d rows, want 1 (it should not be cleared)", migrations)
	}

	if err := client.DropAllTables(ctx); err != nil {
		t.Fatalf("DropAllTables: %v", err)
	}
	if err := client.QueryRow(ctx, `SELECT COUNT(*) FROM django_migrations`).Scan(&migrations); err == nil {
		t.Error("expected django_migrations to be gone after DropAllTables")
	}
}
