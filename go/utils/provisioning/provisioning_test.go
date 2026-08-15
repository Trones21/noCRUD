package provisioning_test

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/provisioning"
)

// ============================================================================
//  Port allocation
// ============================================================================
//
// You can't test a race by running it — that's what makes it a race.
// Reproducing the collision means winning a scheduling lottery, and a test that
// fails one run in a hundred is worse than no test.
//
// So these don't try. They test the two properties that decide whether the race
// can hurt you, and both are deterministic:
//
//  1. A reserved port cannot be taken. The old FindOpenPort returned a port it
//     no longer held, so this property was simply false for it — no scheduling
//     luck involved.
//  2. Losing the port is loud. Force the collision by squatting on the port and
//     require the runner to refuse to start, rather than running the flow
//     against whatever else is listening.

func TestReservedPortCannotBeTaken(t *testing.T) {
	// The invariant the old FindOpenPort could not offer. Between it returning
	// and the app binding, the migrate path spends seconds creating a database
	// and running migrations — and for all of those seconds the port was free
	// for the taking.
	reservation, err := provisioning.ReserveOpenPort()
	if err != nil {
		t.Fatalf("ReserveOpenPort: %v", err)
	}
	defer reservation.Release()

	l, err := net.Listen("tcp", ":"+strconv.Itoa(reservation.Port))
	if err == nil {
		l.Close()
		t.Fatalf("port %d was bindable while reserved — the reservation is not holding it", reservation.Port)
	}
}

func TestReleasedPortIsImmediatelyBindable(t *testing.T) {
	// Holding the port is only useful if the handover still works — a
	// reservation that couldn't be handed over would be a deadlock with extra
	// steps.
	reservation, err := provisioning.ReserveOpenPort()
	if err != nil {
		t.Fatalf("ReserveOpenPort: %v", err)
	}
	reservation.Release()

	l, err := net.Listen("tcp", ":"+strconv.Itoa(reservation.Port))
	if err != nil {
		t.Fatalf("port %d was not bindable after release: %v", reservation.Port, err)
	}
	l.Close()

	// Release is idempotent, so it is safe both to defer and to call at handover.
	reservation.Release()
}

// Worth knowing: this one passes without the fix too. Simultaneous allocation
// was never the problem — while one socket holds a port the OS won't hand it to
// another, so callers that overlap are safe. The danger was always the gap
// after the old helper let go, which is what the two tests above pin down.
func TestConcurrentReservationsAreUnique(t *testing.T) {
	const n = 50
	var mu sync.Mutex
	var reservations []*provisioning.PortReservation

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := provisioning.ReserveOpenPort()
			if err != nil {
				return
			}
			mu.Lock()
			reservations = append(reservations, r)
			mu.Unlock()
		}()
	}
	wg.Wait()

	seen := map[int]bool{}
	for _, r := range reservations {
		if seen[r.Port] {
			t.Errorf("port %d handed out twice", r.Port)
		}
		seen[r.Port] = true
		r.Release()
	}
	if len(reservations) != n {
		t.Errorf("got %d reservations, want %d", len(reservations), n)
	}
}

func TestFindOpenPortDoesNotHoldItsPort(t *testing.T) {
	// Documents what the deprecated helper actually guarantees: nothing. The
	// port is free when it returns — free for the caller, and equally free for
	// everybody else.
	port, err := provisioning.FindOpenPort()
	if err != nil {
		t.Fatalf("FindOpenPort: %v", err)
	}
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("unexpected: FindOpenPort left port %d held", port)
	}
	l.Close()
}

func TestStolenPortIsDetected(t *testing.T) {
	// The collision, forced. Before this check the squatter answered the
	// readiness probe, provisioning returned happily, and the flow ran every
	// request against another flow's app and another flow's database. It
	// usually still passed, which is the worst part: the run was
	// cross-contaminated and nothing said so.
	reservation, err := provisioning.ReserveOpenPort()
	if err != nil {
		t.Fatalf("ReserveOpenPort: %v", err)
	}
	port := reservation.Port
	reservation.Release()

	squatter, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("could not squat on port %d: %v", port, err)
	}
	defer squatter.Close()

	err = provisioning.AssertPortNotTaken(port)
	if err == nil {
		t.Fatalf("port %d was taken by another process, but the runner was willing to "+
			"start a flow on it", port)
	}
	if !strings.Contains(err.Error(), strconv.Itoa(port)) {
		t.Errorf("error should name the port: %v", err)
	}
}

func TestFreePortPassesTheCheck(t *testing.T) {
	// The check must not fire on the normal case.
	reservation, err := provisioning.ReserveOpenPort()
	if err != nil {
		t.Fatalf("ReserveOpenPort: %v", err)
	}
	reservation.Release()

	if err := provisioning.AssertPortNotTaken(reservation.Port); err != nil {
		t.Errorf("AssertPortNotTaken on a free port = %v, want nil", err)
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

// ============================================================================
//  Backend startup
// ============================================================================

func TestWaitForBackendAcceptsAServerThatComesUp(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// A live process, and something listening: the happy path. The detection
	// must not cry wolf.
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	env := &nocrud.Env{AppPort: l.Addr().(*net.TCPAddr).Port, Proc: cmd}
	env.WatchProcess()

	if err := provisioning.WaitForBackend(env, 3*time.Second, nil); err != nil {
		t.Errorf("WaitForBackend = %v, want nil", err)
	}
}

func TestWaitForBackendReportsABackendThatDied(t *testing.T) {
	// Losing the port is only one reason a backend dies on startup — bad
	// settings and missing migrations do it too. Before, any of them meant
	// sitting through the full timeout to be told the server never listened.
	reservation, err := provisioning.ReserveOpenPort()
	if err != nil {
		t.Fatal(err)
	}
	reservation.Release()

	cmd := exec.Command("false") // exits immediately, like a backend that can't start
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	env := &nocrud.Env{AppPort: reservation.Port, Proc: cmd}
	env.WatchProcess()
	env.WaitForExit(5 * time.Second) // deterministic: it is dead before we look

	started := time.Now()
	err = provisioning.WaitForBackend(env, 10*time.Second, nil)
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("expected the dead backend to be reported")
	}
	if !strings.Contains(err.Error(), "exited") {
		t.Errorf("error = %q, want it to say the backend died", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("took %s — it waited out the timeout instead of noticing", elapsed)
	}
}

// Startup watches for the process to exit and so does shutdown; exactly one
// goroutine may Wait on a process, so both have to go through Env.
func TestProcessWatchingIsSharedNotDuplicated(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	env := &nocrud.Env{AppPort: 1, Proc: cmd}
	env.WatchProcess()
	env.WatchProcess() // must not start a second waiter

	if !env.WaitForExit(5 * time.Second) {
		t.Fatal("process did not exit")
	}
	exited, _ := env.ProcessExited()
	if !exited {
		t.Error("ProcessExited() = false after the process ended")
	}

	// Reading the outcome repeatedly is fine.
	for i := 0; i < 3; i++ {
		if exited, _ := env.ProcessExited(); !exited {
			t.Error("ProcessExited() flipped back to false")
		}
	}
}

func TestProcessHelpersAreSafeWithNoProcess(t *testing.T) {
	env := &nocrud.Env{AppPort: 1}
	env.WatchProcess()

	if exited, err := env.ProcessExited(); exited || err != nil {
		t.Errorf("ProcessExited() = %v, %v — want false, nil with no process", exited, err)
	}
	if !env.WaitForExit(time.Millisecond) {
		t.Error("WaitForExit should not block when there is no process")
	}
}
