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
import math
import sys
from pathlib import Path
from statistics import mean

PERF_ROOT = Path(__file__).resolve().parent / "perf"
RUNS_DIR = PERF_ROOT / "runs"
BASELINE_PATH = PERF_ROOT / "baseline.ndjson"

METRICS = ("mean", "p95", "p99")


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


def _percentile(sorted_vals, p):
    """Nearest-rank percentile — robust for any sample count, including n=1."""
    n = len(sorted_vals)
    if n == 0:
        return None
    if n == 1:
        return sorted_vals[0]
    idx = min(max(math.ceil(p / 100.0 * n) - 1, 0), n - 1)
    return sorted_vals[idx]


def aggregate(records):
    """Group samples by (flow, op, endpoint) → count/mean/min/max/p95/p99 in ms."""
    groups = {}
    for r in records:
        k = (r.get("flow"), r.get("op"), r.get("endpoint"))
        groups.setdefault(k, []).append(r["ms"])
    agg = {}
    for k, samples in groups.items():
        s = sorted(samples)
        agg[k] = {
            "count": len(s),
            "mean": mean(s),
            "min": s[0],
            "max": s[-1],
            "p95": _percentile(s, 95),
            "p99": _percentile(s, 99),
        }
    return agg


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


def compare_and_print(run_dir, threshold_pct=20.0, metric="mean"):
    """Print a current-vs-baseline table for `metric` and return regressions.

    Regression detection is on the chosen metric (mean/p95/p99). Percentiles only
    become meaningful once a key has many samples; with one sample per key (plain
    CRUD) p95/p99 equal that sample, and a note is printed to that effect."""
    if metric not in METRICS:
        raise ValueError(f"metric must be one of {METRICS}, got {metric!r}")
    current = aggregate(load_records(run_dir))
    baseline = aggregate(load_records(BASELINE_PATH))

    print(f"\n=== Perf report: {Path(run_dir).name}  (metric: {metric}) ===")
    if not baseline:
        print("No baseline yet. Set one with:  python perf_report.py --set-baseline")
    if not current:
        print("No timings in this run. Did you run flows with --perf?\n")
        return []

    label_w = max(len(_key_label(k)) for k in current)
    header = (
        f"{'key'.ljust(label_w)}  {'n':>3}  "
        f"{'base ms':>9}  {'now ms':>9}  {'delta':>8}"
    )
    print(header)
    print("-" * len(header))

    regressions = []
    for k in sorted(current):
        n = current[k]["count"]
        now = current[k][metric]
        base = baseline.get(k, {}).get(metric)
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
            f"{_key_label(k).ljust(label_w)}  {n:>3}  "
            f"{base_str:>9}  {now:>9.1f}  {delta_str:>8}{flag}"
        )

    if metric in ("p95", "p99") and all(v["count"] < 20 for v in current.values()):
        print(
            f"\nℹ️  Few samples per key — {metric} is noisy here. Run a flow many "
            "times (or use a load flow) for stable percentiles."
        )

    if regressions:
        print(f"\n🔴 {len(regressions)} regression(s) over {threshold_pct:.0f}% ({metric}):")
        for k, base, now, delta in regressions:
            print(f"   {_key_label(k)}: {base:.1f} → {now:.1f} ms ({delta:+.1f}%)")
    else:
        print(f"\n✅ No {metric} regressions over {threshold_pct:.0f}% threshold.")
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
        "--metric",
        choices=METRICS,
        default="mean",
        help="Which metric drives the comparison and gate (default: mean)",
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

    regressions = compare_and_print(run_dir, args.threshold, args.metric)
    sys.exit(1 if regressions else 0)


if __name__ == "__main__":
    main()
