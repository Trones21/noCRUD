#!/usr/bin/env python3
"""
Compare persisted request timings across runs to catch performance regressions.

Timings are produced by running flows with the --perf flag on noCRUD.py, which
writes perf/runs/<run_id>/<flow>.ndjson. This script aggregates a run by
(flow, op, endpoint) and compares it against a committed baseline, flagging any
key whose mean latency regressed beyond a threshold. Exits non-zero when a
regression is found, so it can gate a CI job.

Usage:
    python perf_report.py                  # compare latest run vs baseline
    python perf_report.py --run <run_id>   # compare a specific run
    python perf_report.py --threshold 25   # regression threshold in percent (default 20)
    python perf_report.py --set-baseline   # promote the latest (or --run) to baseline
"""

import argparse
import json
import sys
from pathlib import Path
from statistics import mean

PERF_ROOT = Path(__file__).resolve().parent / "perf"
RUNS_DIR = PERF_ROOT / "runs"
BASELINE_PATH = PERF_ROOT / "baseline.ndjson"


def load_records(path_or_dir):
    """Load NDJSON records from a file, or from every *.ndjson in a directory."""
    p = Path(path_or_dir)
    if not p.exists():
        return []
    files = [p] if p.is_file() else sorted(p.glob("*.ndjson"))
    records = []
    for f in files:
        with open(f) as fh:
            for line in fh:
                line = line.strip()
                if line:
                    records.append(json.loads(line))
    return records


def aggregate(records):
    """Group samples by (flow, op, endpoint) → count/mean/min/max in ms."""
    groups = {}
    for r in records:
        k = (r.get("flow"), r.get("op"), r.get("endpoint"))
        groups.setdefault(k, []).append(r["ms"])
    return {
        k: {"count": len(s), "mean": mean(s), "min": min(s), "max": max(s)}
        for k, s in groups.items()
    }


def latest_run_dir():
    if not RUNS_DIR.exists():
        return None
    runs = [d for d in RUNS_DIR.iterdir() if d.is_dir()]
    return max(runs, key=lambda d: d.name) if runs else None


def resolve_run_dir(run_id):
    if run_id:
        d = RUNS_DIR / run_id
        return d if d.exists() else None
    return latest_run_dir()


def _key_label(k):
    flow, op, endpoint = k
    return f"{flow} · {op} · {endpoint}"


def compare_and_print(run_dir, threshold_pct=20.0):
    """Print a current-vs-baseline table and return the list of regressions."""
    current = aggregate(load_records(run_dir))
    baseline = aggregate(load_records(BASELINE_PATH))

    print(f"\n=== Perf report: {Path(run_dir).name} ===")
    if not baseline:
        print("No baseline yet. Set one with:  python perf_report.py --set-baseline")
    if not current:
        print("No timings in this run. Did you run flows with --perf?\n")
        return []

    label_w = max(len(_key_label(k)) for k in current)
    header = f"{'key'.ljust(label_w)}  {'base ms':>9}  {'now ms':>9}  {'delta':>8}"
    print(header)
    print("-" * len(header))

    regressions = []
    for k in sorted(current):
        now = current[k]["mean"]
        base = baseline.get(k, {}).get("mean")
        if base is None:
            base_str, delta_str, flag = "—", "new", ""
        else:
            delta = ((now - base) / base * 100) if base else 0.0
            base_str, delta_str = f"{base:.1f}", f"{delta:+.1f}%"
            flag = ""
            if delta > threshold_pct:
                regressions.append((k, base, now, delta))
                flag = "  🔴"
        print(
            f"{_key_label(k).ljust(label_w)}  {base_str:>9}  {now:>9.1f}  {delta_str:>8}{flag}"
        )

    if regressions:
        print(f"\n🔴 {len(regressions)} regression(s) over {threshold_pct:.0f}%:")
        for k, base, now, delta in regressions:
            print(f"   {_key_label(k)}: {base:.1f} → {now:.1f} ms ({delta:+.1f}%)")
    else:
        print(f"\n✅ No regressions over {threshold_pct:.0f}% threshold.")
    return regressions


def set_baseline(run_dir):
    records = load_records(run_dir)
    if not records:
        print(f"Nothing to set: no records in {run_dir}")
        return
    PERF_ROOT.mkdir(parents=True, exist_ok=True)
    with open(BASELINE_PATH, "w") as f:
        for r in records:
            f.write(json.dumps(r) + "\n")
    print(
        f"Baseline set from {Path(run_dir).name} "
        f"({len(records)} records) → {BASELINE_PATH}"
    )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run", help="Run id to use (default: latest)")
    parser.add_argument(
        "--threshold",
        type=float,
        default=20.0,
        help="Regression threshold in percent (default: 20)",
    )
    parser.add_argument(
        "--set-baseline",
        action="store_true",
        help="Promote the selected run to the baseline instead of comparing",
    )
    args = parser.parse_args()

    run_dir = resolve_run_dir(args.run)
    if run_dir is None:
        print("No perf runs found. Run flows with:  python noCRUD.py -crud --perf")
        sys.exit(2)

    if args.set_baseline:
        set_baseline(run_dir)
        return

    regressions = compare_and_print(run_dir, args.threshold)
    sys.exit(1 if regressions else 0)


if __name__ == "__main__":
    main()
