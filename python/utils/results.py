"""Machine-readable record of what a run did.

The terminal summary is for you; this file is for everything else — a CI step
that wants to fail on a specific flow, a dashboard, or the anonymized community
submission the nocrud-share-results skill builds.

The schema is shared with the Go runner (`go/utils/results`), the same way perf
records are, so either runner's output can be read by the same tooling.

This file is LOCAL and is not anonymized
----------------------------------------
It deliberately contains flow names, which are usually your endpoint names and
therefore say something about your domain model. That is what makes it useful
locally, and it is exactly why it is not the thing you upload anywhere. The
share skill builds a separate payload from a strict field allowlist and shows it
to you before anything leaves your machine.
"""

import json
import os
import re
import subprocess
from datetime import datetime, timezone

# Bump when the shape changes, so consumers can tell which one they're holding.
SCHEMA = "nocrud.results/v1"

_ANSI = re.compile(r"\x1b\[[0-9;]*m")


def strip_ansi(text: str) -> str:
    """Remove terminal colour codes. They belong on a terminal, not in JSON."""
    return _ANSI.sub("", str(text))


def flow_record(name: str, kind: str, summary, ms: float) -> dict:
    """One flow's outcome, normalized.

    A flow fails by raising, and both runners turn that into a summary starting
    with "Fail:" — so that prefix, not the presence of a ✘, is what decides `ok`.
    A CRUD flow that reports U:✘ did not raise: the update comparison came back
    false, which is a real finding but not an error. The ticks stay visible in
    `summary` either way.
    """
    text = strip_ansi(summary)
    return {
        "name": name,
        "kind": kind,
        "ok": not text.startswith("Fail:"),
        "ms": round(ms, 6),
        "summary": text,
    }


def runner_sha() -> str:
    """noCRUD's own commit, for telling which runner version produced a result.

    Not the app-under-test's.
    """
    try:
        here = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        out = subprocess.run(
            ["git", "rev-parse", "--short", "HEAD"],
            cwd=here,
            capture_output=True,
            text=True,
            check=True,
        )
        return out.stdout.strip()
    except Exception:
        return ""


def build(flows, wall_ms, mode, started_at, jobs=0, passed=None) -> dict:
    if passed is None:
        passed = all(f["ok"] for f in flows) if flows else True
    data = {
        "schema": SCHEMA,
        "runner": "python",
        "started_at": started_at,
        "wall_ms": round(wall_ms, 6),
        "mode": mode,
        "passed": passed,
        "flows": flows,
    }
    sha = runner_sha()
    if sha:
        data["runner_sha"] = sha
    if jobs:
        data["jobs"] = jobs
    return data


def write(path: str, data: dict) -> None:
    parent = os.path.dirname(os.path.abspath(path))
    if parent:
        os.makedirs(parent, exist_ok=True)
    with open(path, "w") as fh:
        json.dump(data, fh, indent=2)
        fh.write("\n")


def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
