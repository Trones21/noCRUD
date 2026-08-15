// Package provisioning gives each flow a world of its own — the Go counterpart
// of python/utils/provisioning.py.
//
// Parallel mode is only safe if flows can't see each other's data, so every
// flow gets its own database and its own backend on its own port:
//
//	             ┌───────────────────┐     ┌──────────────────┐
//	        -->  │ App :1 Port 8001  │ --> │ DB: noCRUD_p8001 │
//	       /     ├───────────────────┤     ├──────────────────┤
//	Runner --->  │ App :2 Port 8002  │ --> │ DB: noCRUD_p8002 │
//	       \     ├───────────────────┤     ├──────────────────┤
//	        -->  │ App :3 Port 8003  │ --> │ DB: noCRUD_p8003 │
//	             └───────────────────┘     └──────────────────┘
//
// ProvisionEnvForFlow is the seam to customise per project. The three prebuilt
// implementations below differ only in how the schema gets into the new
// database; migrate is the one that needs no setup.
package provisioning

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Trones21/noCRUD/go/config"
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/dbclient"
)

// BackendStartTimeout is how long to wait for the app to start listening.
var BackendStartTimeout = 30 * time.Second

// ProvisionEnvForFlow provisions the environment for one flow. Customise this
// per project.
//
// It is expected to:
//   - create an isolated DB
//   - run migrations or load the schema
//   - start the backend on a unique port
//
// Which prebuilt implementation runs is chosen by NOCRUD_PROVISION
// (migrate | sql | template), defaulting to migrate. If your backend isn't
// Django, replace the body of this function — everything above it in the
// runner is framework-agnostic HTTP.
func ProvisionEnvForFlow(ctx context.Context, flowName string, out io.Writer) (*nocrud.Env, error) {
	switch strings.ToLower(os.Getenv(config.EnvProvision)) {
	case "sql":
		return ProvisionDjangoEnvDirectViaSQL(ctx, flowName, out)
	case "template":
		return ProvisionDjangoEnvDirectViaTemplateDB(ctx, flowName, out)
	default:
		return ProvisionDjangoEnvUsingMigrate(ctx, flowName, out)
	}
}

// =================================================
//  Pre-built Provision Funcs
// =================================================

// ProvisionDjangoEnvUsingMigrate creates the database and runs manage.py
// migrate into it. Slowest of the three, but needs no setup at all.
func ProvisionDjangoEnvUsingMigrate(ctx context.Context, flowName string, out io.Writer) (*nocrud.Env, error) {
	env, err := newEnv(ctx, flowName, out)
	if err != nil {
		return nil, err
	}

	// Django's makemigrations has no --noinput, so it hangs whenever it wants
	// input (which it frequently does). Migration files therefore have to
	// already exist — we check rather than generate.
	if err := EnsureMigrationsExist(config.AppDir()); err != nil {
		return env, err
	}
	fmt.Fprintln(out, "🛡️  MIGRATION CHECK PASSED — Found existing migration files.")

	if err := RunMgmtCommandQuietly(ctx, env, "migrate"); err != nil {
		return env, err
	}
	fmt.Fprintf(out, "✅  Migration succeeded for DB: %s\n", env.DBName)

	if err := DBMatchCheck(ctx, env, out); err != nil {
		return env, err
	}
	return env, StartBackend(ctx, env, out)
}

// ProvisionDjangoEnvDirectViaSQL creates the database and loads a schema dump
// into it with psql — much faster than migrating, at the cost of keeping
// schema.sql current.
func ProvisionDjangoEnvDirectViaSQL(ctx context.Context, flowName string, out io.Writer) (*nocrud.Env, error) {
	env, err := newEnv(ctx, flowName, out)
	if err != nil {
		return nil, err
	}

	schemaPath := filepath.Join(config.AppDir(), "schema.sql")
	if err := EnsureSchemaDefinitionExists(schemaPath); err != nil {
		return env, err
	}
	fmt.Fprintf(out, "📄 Schema file found: %s\n", schemaPath)

	cmd := exec.CommandContext(ctx, "psql",
		"-U", env.DBConfig.User,
		"-h", env.DBConfig.Host,
		"-p", env.DBConfig.Port,
		"-d", env.DBName,
		"-f", schemaPath,
	)
	cmd.Env = env.Environ()
	cmd.Stdout = io.Discard
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return env, fmt.Errorf("loading schema into %s: %w", env.DBName, err)
	}

	if err := DBMatchCheck(ctx, env, out); err != nil {
		return env, err
	}
	return env, StartBackend(ctx, env, out)
}

// ProvisionDjangoEnvDirectViaTemplateDB clones an already-migrated template
// database. The fastest option, and the one that needs the most setup: build
// the template once before the run.
func ProvisionDjangoEnvDirectViaTemplateDB(ctx context.Context, flowName string, out io.Writer) (*nocrud.Env, error) {
	template := os.Getenv("NOCRUD_TEMPLATE_DB")
	if template == "" {
		template = "template_db"
	}

	port, err := FindOpenPort()
	if err != nil {
		return nil, err
	}
	env := envForPort(flowName, port)

	admin, err := dbclient.New(ctx, dbclient.AdminConfig(), out)
	if err != nil {
		return env, err
	}
	env.Admin = admin

	if err := ProvisionDBFromTemplate(ctx, env, template, out); err != nil {
		return env, err
	}
	if err := DBMatchCheck(ctx, env, out); err != nil {
		return env, err
	}
	return env, StartBackend(ctx, env, out)
}

// =================================================
//  Helpers
// =================================================

// newEnv picks a port, names the database after it, and creates it.
func newEnv(ctx context.Context, flowName string, out io.Writer) (*nocrud.Env, error) {
	port, err := FindOpenPort()
	if err != nil {
		return nil, err
	}
	env := envForPort(flowName, port)

	admin, err := dbclient.New(ctx, dbclient.AdminConfig(), out)
	if err != nil {
		return env, err
	}
	env.Admin = admin

	if err := admin.CreateDB(ctx, env.DBName); err != nil {
		return env, err
	}
	return env, nil
}

// envForPort derives a flow's environment from its port, so the database name
// says which app instance owns it.
func envForPort(flowName string, port int) *nocrud.Env {
	dbName := fmt.Sprintf("noCRUD_p%d_%s", port, flowName)
	cfg := dbclient.ConfigFromEnv()
	cfg.DBName = dbName
	return &nocrud.Env{
		DBName:   dbName,
		AppPort:  port,
		DBConfig: cfg,
	}
}

// FindOpenPort finds an available port by letting the OS assign one.
func FindOpenPort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// RunMgmtCommandQuietly runs a manage.py command, surfacing stderr only when it
// fails.
func RunMgmtCommandQuietly(ctx context.Context, env *nocrud.Env, args ...string) error {
	cmd := exec.CommandContext(ctx, "python", append([]string{"manage.py"}, args...)...)
	cmd.Dir = config.AppDir()
	cmd.Env = env.Environ()
	cmd.Stdin = nil
	cmd.Stdout = io.Discard

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %s\nStderr:\n%s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return nil
}

// EnsureSchemaDefinitionExists checks that a schema dump is present.
func EnsureSchemaDefinitionExists(schemaPath string) error {
	if _, err := os.Stat(schemaPath); err != nil {
		return fmt.Errorf(
			"❌ Schema definition not found. Expected at: %s\n"+
				"Please generate it using:\n"+
				"    pg_dump -U postgres -h 0.0.0.0 --schema-only --no-owner --no-privileges <db_to_dump> > schema.sql\n"+
				"Or switch to a provision method that uses Django migrations.", schemaPath)
	}
	return nil
}

// EnsureMigrationsExist checks that each app has at least one migration beyond
// __init__.py.
func EnsureMigrationsExist(appsDir string) error {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return fmt.Errorf("reading app dir %s: %w", appsDir, err)
	}

	var missing []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		migrationsDir := filepath.Join(appsDir, entry.Name(), "migrations")
		info, err := os.Stat(migrationsDir)
		if err != nil || !info.IsDir() {
			continue
		}
		files, err := filepath.Glob(filepath.Join(migrationsDir, "*.py"))
		if err != nil {
			return err
		}
		found := false
		for _, f := range files {
			if filepath.Base(f) != "__init__.py" {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, entry.Name())
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing migration files in: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ProvisionDBFromTemplate creates a database by cloning a template with
// createdb -T.
func ProvisionDBFromTemplate(ctx context.Context, env *nocrud.Env, template string, out io.Writer) error {
	cmd := exec.CommandContext(ctx, "createdb",
		"-h", env.DBConfig.Host,
		"-p", env.DBConfig.Port,
		"-U", env.DBConfig.User,
		"-T", template,
		env.DBName,
	)
	cmd.Env = env.Environ()
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ failed to create DB %s from template %s: %w", env.DBName, template, err)
	}
	fmt.Fprintf(out, "✅ Created DB from template: %s\n", env.DBName)
	return nil
}

// DBMatchCheck confirms the backend will use the database the runner just
// created. Getting this wrong is the single most common setup mistake, and
// without the check it shows up much later as baffling test failures.
func DBMatchCheck(ctx context.Context, env *nocrud.Env, out io.Writer) error {
	err := dbMatchCheckViaSettings(ctx, env, out)
	if err == nil {
		return nil
	}
	fmt.Fprintf(out,
		"🔄 Fast DB match check aborted: could not read settings.py.\n"+
			" Falling back to slower DB match check (manage.py shell)\n"+
			" Inner exception was:\n→ %v\n", err)
	return dbMatchCheckSlow(ctx, env, out)
}

// dbMatchCheckViaSettings executes settings.py standalone and reads the
// resolved database name out of it — much faster than booting Django.
func dbMatchCheckViaSettings(ctx context.Context, env *nocrud.Env, out io.Writer) error {
	const script = `import runpy, sys; print(runpy.run_path(sys.argv[1])["DATABASES"]["default"]["NAME"])`

	cmd := exec.CommandContext(ctx, "python", "-c", script, config.SettingsPath())
	cmd.Dir = config.AppDir()
	cmd.Env = env.Environ()
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running %s: %w", config.SettingsPath(), err)
	}

	name := lastLine(string(output))
	if name != env.DBName {
		return fmt.Errorf("[SETTINGS MISMATCH] settings.py resolves DB name to '%s', but runner created '%s'", name, env.DBName)
	}
	fmt.Fprintf(out, "✅ settings.py resolves DB to expected value: %s\n", name)
	return nil
}

// dbMatchCheckSlow asks Django itself, via manage.py shell.
func dbMatchCheckSlow(ctx context.Context, env *nocrud.Env, out io.Writer) error {
	cmd := exec.CommandContext(ctx, "python", "manage.py", "shell", "-c",
		"from django.db import connection; print(connection.settings_dict['NAME'])")
	cmd.Dir = config.AppDir()
	cmd.Env = env.Environ()

	var stderr strings.Builder
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to query current DB:\n%s", strings.TrimSpace(stderr.String()))
	}

	inUse := lastLine(string(output))
	if inUse != env.DBName {
		return fmt.Errorf(
			"[DB MISMATCH] Django is using DB '%s', expected to use the db created by the runner '%s'.\n"+
				"            Please ensure that provisioning and settings.py use the same environment variable for the database name.\n"+
				"            Settings.py MUST use an environment variable because the database name is programmatically generated",
			inUse, env.DBName)
	}
	fmt.Fprintf(out, "✅ Django is using expected DB: %s\n", inUse)
	return nil
}

// StartBackend starts the app and blocks until it is listening.
//
// Its log lines go straight to the real stdout, prefixed with the port, rather
// than into the flow's buffer — in parallel mode you want to see the backend
// while the flow is still running, and the port says which one is talking.
func StartBackend(ctx context.Context, env *nocrud.Env, out io.Writer) error {
	cmd := exec.CommandContext(ctx, "python", "manage.py", "runserver", "--noreload",
		fmt.Sprintf("0.0.0.0:%d", env.AppPort))
	cmd.Dir = config.AppDir()
	cmd.Env = env.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting backend on port %d: %w", env.AppPort, err)
	}
	env.Proc = cmd

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Fprintf(os.Stdout, "[Django:%d] %s\n", env.AppPort, scanner.Text())
		}
	}()

	return WaitForBackendToListen(env.AppPort, BackendStartTimeout)
}

// WaitForBackendToListen blocks until something is listening on the port.
func WaitForBackendToListen(port int, timeout time.Duration) error {
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 250*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for server to listen on port %d", port)
}

// CleanupEnv stops the backend and drops the database.
//
// Every step runs even if an earlier one failed: a warning here should never
// mask the flow's own result, and a leaked backend process is worse than a
// noisy log line.
func CleanupEnv(ctx context.Context, env *nocrud.Env, out io.Writer) {
	if env == nil {
		return
	}

	if env.Proc != nil && env.Proc.Process != nil {
		if err := stopProcess(env.Proc); err != nil {
			fmt.Fprintf(out, "⚠️ Cleanup warning: stopping backend: %v\n", err)
		}
	}

	if env.Admin != nil {
		if env.PersistDB {
			fmt.Fprintf(out, "Persisting db %s\n", env.DBName)
		} else if err := env.Admin.DropDB(ctx, env.DBName); err != nil {
			fmt.Fprintf(out, "⚠️ Cleanup warning: %v\n", err)
		}
		if err := env.Admin.Close(ctx); err != nil {
			fmt.Fprintf(out, "⚠️ Cleanup warning: closing admin connection: %v\n", err)
		}
	}
}

// stopProcess asks the backend to exit, and insists if it won't. Waiting
// matters: the database can't be dropped while the app still holds a
// connection to it.
func stopProcess(cmd *exec.Cmd) error {
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = cmd.Process.Kill()
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			return err
		}
		<-done
		return nil
	}
}

// lastLine returns the final non-empty line of output — Django likes to print
// warnings before the value you asked for.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}
