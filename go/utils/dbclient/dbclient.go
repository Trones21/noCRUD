// Package dbclient is the back door to the database — the Go counterpart of
// python/utils/db_client.py.
//
// Flows should prefer the API (that's the whole point of noCRUD: exercise the
// backend the way a user does). This is here for the things the API can't do:
// creating and dropping the per-flow database, and resetting state between
// serial runs.
package dbclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Trones21/noCRUD/go/config"
	"github.com/Trones21/noCRUD/go/utils/fixtures"
)

// pgDuplicateDatabase is SQLSTATE 42P04 — the "already exists" we tolerate.
const pgDuplicateDatabase = "42P04"

// Config is a postgres connection.
type Config struct {
	DBName string
	User   string
	Pass   string
	Host   string
	Port   string
}

// ConfigFromEnv reads the connection from the environment, the way the app
// under test does (DB_NAME, DB_USER, DB_PASS, DB_HOST, DB_PORT).
func ConfigFromEnv() Config {
	return Config{
		DBName: envOr("DB_NAME", "postgres"),
		User:   envOr("DB_USER", "postgres"),
		Pass:   envOr("DB_PASS", "postgres"),
		Host:   envOr("DB_HOST", "localhost"),
		Port:   envOr("DB_PORT", "5432"),
	}
}

// AdminConfig connects as the superuser to the maintenance database, which is
// what creating and dropping per-flow databases requires.
func AdminConfig() Config {
	cfg := ConfigFromEnv()
	cfg.DBName = "postgres"
	cfg.User = envOr("DB_ADMIN_USER", "postgres")
	cfg.Pass = envOr("DB_ADMIN_PASS", "postgres")
	return cfg
}

// URL renders the config as a postgres connection string.
func (c Config) URL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.User, c.Pass, c.Host, c.Port, c.DBName)
}

// Client is one connection to postgres.
type Client struct {
	Config Config

	conn *pgx.Conn
	out  io.Writer
}

// New opens a connection. out is where progress is logged; nil means stdout.
func New(ctx context.Context, cfg Config, out io.Writer) (*Client, error) {
	if out == nil {
		out = os.Stdout
	}
	conn, err := pgx.Connect(ctx, cfg.URL())
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres at %s:%s/%s as %s: %w",
			cfg.Host, cfg.Port, cfg.DBName, cfg.User, err)
	}
	return &Client{Config: cfg, conn: conn, out: out}, nil
}

// Close releases the connection.
func (c *Client) Close(ctx context.Context) error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close(ctx)
}

// Exec runs a statement. The Python client leaves its connection and cursor
// public for whatever a project needs beyond the methods here; these three are
// the same escape hatch.
func (c *Client) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := c.conn.Exec(ctx, sql, args...)
	return err
}

// QueryRow runs a query expected to return one row.
func (c *Client) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return c.conn.QueryRow(ctx, sql, args...)
}

// Conn exposes the underlying connection.
func (c *Client) Conn() *pgx.Conn { return c.conn }

// CreateDB creates a database, treating "already exists" as a warning rather
// than a failure.
func (c *Client) CreateDB(ctx context.Context, name string) error {
	_, err := c.conn.Exec(ctx, "CREATE DATABASE "+quote(name))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgDuplicateDatabase {
			fmt.Fprintf(c.out, "⚠️ Database '%s' already exists.\n", name)
			return nil
		}
		fmt.Fprintf(c.out, "❌ Failed to create database '%s': %v\n", name, err)
		return err
	}
	fmt.Fprintf(c.out, "✅ Database '%s' created.\n", name)
	return nil
}

// DropDB drops a database if it exists.
//
// Any leftover connection to it makes the drop fail, so lingering sessions are
// terminated first. The backend process is normally already gone by this point;
// this covers the case where it isn't.
func (c *Client) DropDB(ctx context.Context, name string) error {
	if err := c.TerminateConnections(ctx, name); err != nil {
		fmt.Fprintf(c.out, "⚠️ Could not terminate sessions on '%s': %v\n", name, err)
	}
	if _, err := c.conn.Exec(ctx, "DROP DATABASE IF EXISTS "+quote(name)); err != nil {
		fmt.Fprintf(c.out, "❌ Failed to drop database '%s': %v\n", name, err)
		return err
	}
	fmt.Fprintf(c.out, "🗑️  Database '%s' dropped.\n", name)
	return nil
}

// TerminateConnections kicks every other session off a database.
func (c *Client) TerminateConnections(ctx context.Context, name string) error {
	_, err := c.conn.Exec(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity
		 WHERE datname = $1 AND pid <> pg_backend_pid()`, name)
	return err
}

// ClearTable deletes every row from one table.
func (c *Client) ClearTable(ctx context.Context, table string) error {
	_, err := c.conn.Exec(ctx, "DELETE FROM "+quote(table))
	return err
}

// appTables lists the public tables that belong to the app — everything except
// Django's own bookkeeping, which we never want to wipe.
func (c *Client) appTables(ctx context.Context) ([]string, error) {
	rows, err := c.conn.Query(ctx, `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
			AND tablename NOT LIKE 'django_%'
			AND tablename NOT LIKE 'auth_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

// ClearAllTables empties every app table without dropping them.
//
// Foreign keys mean the order matters and isn't known up front, so this makes
// several passes and lets the ones that aren't ready yet fail — by the last
// pass their dependents are gone.
func (c *Client) ClearAllTables(ctx context.Context) error {
	tables, err := c.appTables(ctx)
	if err != nil {
		return err
	}
	for pass := 0; pass < 3; pass++ {
		for _, table := range tables {
			_, _ = c.conn.Exec(ctx, "DELETE FROM "+quote(table)+" CASCADE")
		}
	}
	return nil
}

// VerifyAllTablesCleared checks that ClearAllTables actually got everything.
//
// The Python version exits the process here; returning an error instead lets
// the runner attribute the failure to the flow that caused it, which matters
// when the other flows are still running.
func (c *Client) VerifyAllTablesCleared(ctx context.Context) error {
	tables, err := c.appTables(ctx)
	if err != nil {
		return err
	}
	for _, table := range tables {
		var count int
		if err := c.conn.QueryRow(ctx, "SELECT COUNT(*) FROM "+quote(table)).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			fmt.Fprintf(c.out, "⚠️ Table %s still has %d records!\n", table, count)
			return fmt.Errorf("table %s still has %d records after clearing", table, count)
		}
	}
	return nil
}

// DropAllTables drops every table in the public schema.
func (c *Client) DropAllTables(ctx context.Context) error {
	_, err := c.conn.Exec(ctx, `
		DO $$
		DECLARE
			r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
				EXECUTE 'DROP TABLE IF EXISTS public.' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;`)
	if err != nil {
		return err
	}
	fmt.Fprintln(c.out, "tables dropped")
	return nil
}

// Reset drops all tables, re-runs migrate, and loads fixtures. The order of
// fixtures matters. Passing none loads the minimal set.
func (c *Client) Reset(ctx context.Context, env []string, fixtureNames ...string) error {
	if err := c.DropAllTables(ctx); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "python", "manage.py", "migrate")
	cmd.Dir = config.AppDir()
	cmd.Env = env
	cmd.Stdout = c.out
	cmd.Stderr = c.out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("manage.py migrate: %w", err)
	}
	return fixtures.Add(c.out, env, fixtureNames...)
}

// ClearDataExceptMinimalFixtures empties the app tables and reloads fixtures.
// Faster than Reset because no migrations run.
func (c *Client) ClearDataExceptMinimalFixtures(ctx context.Context, env []string, fixtureNames ...string) error {
	if err := c.ClearAllTables(ctx); err != nil {
		return err
	}
	if err := c.VerifyAllTablesCleared(ctx); err != nil {
		return err
	}
	fmt.Fprintln(c.out, "All tables cleared (except django required)")
	return fixtures.Add(c.out, env, fixtureNames...)
}

// WaitReady blocks until postgres accepts connections, so a runner started
// alongside its database (docker compose, CI service container) doesn't race it.
func WaitReady(ctx context.Context, cfg Config, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := pgx.Connect(ctx, cfg.URL())
		if err == nil {
			return conn.Close(ctx)
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("postgres not ready after %s: %w", timeout, lastErr)
}

// quote makes an identifier safe to splice into a statement. Database and table
// names here are generated from flow names, so they are ours — but they still
// go through the sanitizer rather than string concatenation on trust.
func quote(identifier string) string {
	return pgx.Identifier{identifier}.Sanitize()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
