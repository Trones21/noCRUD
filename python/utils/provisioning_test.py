"""Tests for port reservation and backend startup.

Run directly, or with pytest:

    python utils/provisioning_test.py
    pytest utils/provisioning_test.py

## On testing a race condition

You can't test a race by running it — that's what makes it a race. Reproducing
the collision means winning a scheduling lottery, and a test that fails one run
in a hundred is worse than no test.

So these don't try. They test the two properties that decide whether the race
can hurt you, and both are deterministic:

1. **A reserved port cannot be taken.** Reserve one, then try to bind it — that
   must fail, every time. The old `find_open_port()` returned a port it no
   longer held, so this property was simply false for it, and no amount of
   scheduling luck was involved.

2. **Losing the port is loud.** Force the collision by squatting on the port,
   then require the runner to refuse to start. This is the one that matters:
   before, the squatter answered the readiness probe and the flow ran against
   someone else's app and database without a word.

   Note what this test is *not* allowed to rely on. An earlier version of the
   fix only watched our own backend process, on the theory that a backend which
   loses its bind exits. It does — about a second later, after Django has
   finished booting. The squatter answers the probe immediately, so at the
   moment we look, our process is alive and everything appears fine. The check
   has to happen *before* the backend is spawned, when nothing of ours can
   possibly be answering; that is the only version of it that isn't a race in
   its own right.

Together those bound the problem from both ends — the collision becomes rare,
and the remainder becomes an error instead of corrupt data.
"""

import socket
import subprocess
import sys
import threading
import time

from utils.provisioning import (
    assert_port_not_taken,
    find_open_port,
    reserve_open_port,
    wait_for_backend_to_listen_on_port,
)


def bind_attempt(port):
    """Tries to bind port. Returns the OSError, or None if it succeeded."""
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    try:
        s.bind(("0.0.0.0", port))
        return None
    except OSError as e:
        return e
    finally:
        s.close()


def spawn_listener(port):
    """A stand-in backend that listens until killed."""
    return subprocess.Popen(
        [
            sys.executable,
            "-c",
            "import socket,sys,time;"
            "s=socket.socket();"
            "s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1);"
            f"s.bind(('0.0.0.0',{port}));s.listen(5);"
            "time.sleep(30)",
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )


def spawn_failing_backend(port):
    """A stand-in backend that cannot bind, like Django losing the port."""
    return subprocess.Popen(
        [
            sys.executable,
            "-c",
            "import socket,sys;"
            "s=socket.socket();"
            f"s.bind(('0.0.0.0',{port}))",
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )


# ---------------------------------------------------------------------------
# 1. A reserved port cannot be taken
# ---------------------------------------------------------------------------


def test_reserved_port_cannot_be_taken():
    """The whole point of holding the socket: nobody else can have it.

    This is the invariant the old find_open_port() could not offer. Between it
    returning and Django binding, the migrate path spends seconds creating a
    database and running migrations — and for all of those seconds the port was
    free for the taking.
    """
    port, reservation = reserve_open_port()
    try:
        err = bind_attempt(port)
        assert err is not None, (
            f"port {port} was bindable while reserved — the reservation is not holding it"
        )
    finally:
        reservation.close()
    print("✅ reserved port cannot be taken by anyone else")


def test_released_port_is_immediately_bindable():
    """And once released, the backend can actually take it.

    Holding the port is only useful if the handoff still works — a reservation
    that couldn't be handed over would just be a deadlock with extra steps.
    """
    port, reservation = reserve_open_port()
    reservation.close()

    err = bind_attempt(port)
    assert err is None, f"port {port} was not bindable after release: {err}"
    print("✅ released port is immediately bindable by the backend")


def test_find_open_port_does_not_hold_its_port():
    """Documents why reserve_open_port() exists.

    find_open_port() is kept for runners that already call it, and this pins its
    actual guarantee: none. The port is free when it returns — free for the
    caller, and equally free for everybody else.
    """
    port = find_open_port()
    err = bind_attempt(port)
    assert err is None, "unexpected: find_open_port left the port held"
    print("✅ find_open_port returns an unheld port (which is why it is not used)")


def test_concurrent_reservations_are_unique():
    """Every flow provisioning at once must get a port of its own.

    Worth knowing: this one passes without the fix too. Simultaneous allocation
    was never the problem — while one socket holds a port the OS won't hand it
    to another, so callers that overlap are safe. The danger was always the gap
    after the old helper let go, which is what the two tests above pin down.
    """
    reservations = []
    lock = threading.Lock()

    def reserve():
        pair = reserve_open_port()
        with lock:
            reservations.append(pair)

    threads = [threading.Thread(target=reserve) for _ in range(50)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()

    try:
        ports = [port for port, _ in reservations]
        assert len(set(ports)) == len(ports), (
            f"duplicate ports handed out: {len(ports) - len(set(ports))} collisions"
        )
    finally:
        for _, reservation in reservations:
            reservation.close()
    print(f"✅ {len(reservations)} concurrent reservations, all distinct")


# ---------------------------------------------------------------------------
# 2. Losing the port is loud
# ---------------------------------------------------------------------------


def test_stolen_port_is_detected_before_the_backend_starts():
    """The collision, forced — and it must not pass silently.

    Before this check, the squatter answered the readiness probe, provisioning
    returned happily, and the flow ran every request against another flow's app
    and another flow's database. It usually still *passed*, which is the worst
    part: the run was cross-contaminated and nothing said so.
    """
    port, reservation = reserve_open_port()
    reservation.close()

    squatter = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    squatter.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    squatter.bind(("0.0.0.0", port))
    squatter.listen(5)

    try:
        assert_port_not_taken(port)
        raise AssertionError(
            f"port {port} was taken by another process, but the runner was "
            f"willing to start a flow on it — every request would have gone to "
            f"the other process's app and database"
        )
    except RuntimeError as e:
        assert str(port) in str(e), f"error should name the port: {e}"
    finally:
        squatter.close()
    print("✅ a stolen port is caught before the backend is even spawned")


def test_free_port_passes_the_check():
    """And the check must not fire on the normal case."""
    port, reservation = reserve_open_port()
    reservation.close()

    assert_port_not_taken(port)  # raises on failure
    print("✅ an untaken port passes the check")


def test_backend_that_dies_on_startup_is_reported():
    """A backend that exits should be reported, not waited out.

    Losing the port is only one reason a backend dies on startup — bad settings
    and missing migrations do it too. Before, any of them meant sitting through
    the full timeout to be told the server never listened.
    """
    port, reservation = reserve_open_port()
    reservation.close()

    proc = spawn_failing_backend(port + 1 if port < 65535 else port - 1)
    output = []
    threading.Thread(
        target=lambda: output.extend(proc.stdout.readlines()), daemon=True
    ).start()
    proc.wait(timeout=10)  # make it deterministic: it is dead before we look

    started = time.time()
    try:
        wait_for_backend_to_listen_on_port(
            port, timeout=10, proc=proc, recent_output=output
        )
        raise AssertionError("expected the dead backend to be reported")
    except RuntimeError as e:
        assert str(port) in str(e), f"error should name the port: {e}"
        assert "exited" in str(e), f"error should say the backend died: {e}"

    elapsed = time.time() - started
    assert elapsed < 5, f"took {elapsed:.1f}s — it waited out the timeout instead of noticing"
    print(f"✅ a backend that dies on startup is reported in {elapsed:.2f}s, not after the timeout")


def test_wait_succeeds_when_our_backend_is_listening():
    """The happy path still works — the detection must not cry wolf."""
    port, reservation = reserve_open_port()
    reservation.close()

    proc = spawn_listener(port)
    try:
        assert wait_for_backend_to_listen_on_port(port, timeout=5, proc=proc) is True
    finally:
        proc.kill()
    print("✅ a backend that comes up normally is accepted")


def test_wait_without_a_proc_still_works():
    """Serial mode and older callers pass no process — that path must survive."""
    port, reservation = reserve_open_port()
    reservation.close()

    proc = spawn_listener(port)
    try:
        assert wait_for_backend_to_listen_on_port(port, timeout=5) is True
    finally:
        proc.kill()
    print("✅ the no-process call signature still works")


def test_timeout_when_nothing_ever_listens():
    port, reservation = reserve_open_port()
    reservation.close()

    try:
        wait_for_backend_to_listen_on_port(port, timeout=1)
        raise AssertionError("expected a timeout")
    except TimeoutError as e:
        assert str(port) in str(e)
    print("✅ a backend that never listens times out")


if __name__ == "__main__":
    failures = 0
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            try:
                fn()
            except AssertionError as e:
                failures += 1
                print(f"❌ {name}: {e}")
    print("\nAll port reservation tests passed" if not failures else f"\n{failures} failed")
    sys.exit(1 if failures else 0)
