"""
Persisted request timings for regression tracking.

Off by default — zero behavior change unless enabled. Turn it on with the env var
NOCRUD_PERF=1 (the --perf flag on noCRUD.py does this for you). When enabled,
every timed request (see `with_perf` in decorators.py) is recorded and written to
perf/runs/<run_id>/<flow>.ndjson, so runs can be compared over time — e.g. in CI,
before/after a code change, to see how your API's timings moved.

Records are newline-delimited JSON, one object per timed call:
    {"run_id", "git_sha", "ts", "flow", "op", "endpoint", "ms"}

One file per flow (rather than a single shared file) keeps parallel mode safe:
each flow runs in its own process and writes its own file, so there is no
cross-process write contention.
"""

import json
import os
import subprocess
from datetime import datetime, timezone
from pathlib import Path

PERF_ROOT = Path(__file__).resolve().parents[1] / "perf"
RUNS_DIR = PERF_ROOT / "runs"
BASELINE_PATH = PERF_ROOT / "baseline.ndjson"

# In-process buffer for the flow currently running. In parallel mode each worker
# process has its own copy; in serial mode it is reused and cleared per flow.
_records = []
_current_flow = "unknown"


def enabled() -> bool:
    return os.getenv("NOCRUD_PERF") not in (None, "", "0", "false", "False")


def _git_sha() -> str:
    try:
        return subprocess.run(
            ["git", "rev-parse", "--short", "HEAD"],
            capture_output=True,
            text=True,
            check=True,
        ).stdout.strip()
    except Exception:
        return "unknown"


def run_id() -> str:
    return os.getenv("NOCRUD_PERF_RUN_ID", "adhoc")


def init_run() -> str:
    """Create a run id and resolve the git sha once, exporting both to the
    environment so child processes (parallel mode) share them. Call this once in
    the parent before running any flows. Returns the run id."""
    # Microsecond precision so two runs started in the same second don't collide
    # (fixed-width, so run ids still sort lexicographically by time).
    rid = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S_%fZ")
    os.environ["NOCRUD_PERF_RUN_ID"] = rid
    os.environ.setdefault("NOCRUD_PERF_SHA", _git_sha())
    (RUNS_DIR / rid).mkdir(parents=True, exist_ok=True)
    return rid


def set_current_flow(name: str):
    """Tag subsequently-recorded timings with this flow name."""
    global _current_flow
    _current_flow = name


def record(op: str, endpoint, ms: float):
    """Buffer one timed request. No-op unless perf collection is enabled."""
    if not enabled():
        return
    _records.append(
        {
            "run_id": run_id(),
            "git_sha": os.getenv("NOCRUD_PERF_SHA", "unknown"),
            "ts": datetime.now(timezone.utc).isoformat(),
            "flow": _current_flow,
            "op": op,
            "endpoint": endpoint,
            "ms": round(ms, 3),
        }
    )


def flush_flow(flow_name: str):
    """Write this flow's buffered records to disk and clear the buffer. Safe to
    call unconditionally at a flow boundary."""
    global _records
    if not enabled() or not _records:
        _records = []
        return
    out_dir = RUNS_DIR / run_id()
    out_dir.mkdir(parents=True, exist_ok=True)
    out_path = out_dir / f"{_safe(flow_name)}.ndjson"
    with open(out_path, "a") as f:
        for rec in _records:
            f.write(json.dumps(rec) + "\n")
    _records = []


def _safe(name: str) -> str:
    return "".join(c if c.isalnum() or c in "-_" else "_" for c in name)
