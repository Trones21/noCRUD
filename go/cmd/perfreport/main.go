// Command perfreport compares persisted request timings across runs to catch
// performance regressions — the Go counterpart of python/perf_report.py.
//
// Timings are produced by running flows with --perf, which writes
// perf/runs/<run_id>/<flow>.ndjson. This command aggregates a run by
// (flow, op, endpoint) and compares it against a committed baseline, flagging
// any key whose latency regressed beyond a threshold. It exits non-zero when a
// regression is found, so it can gate a CI job.
//
// Usage:
//
//	go run ./cmd/perfreport                  # compare latest run vs baseline
//	go run ./cmd/perfreport --run <run_id>   # compare a specific run
//	go run ./cmd/perfreport --threshold 25   # regression threshold in percent (default 20)
//	go run ./cmd/perfreport --metric p95     # gate on p95 instead of mean
//	go run ./cmd/perfreport --set-baseline   # promote the latest (or --run) to baseline
//
// The NDJSON it reads is the same schema the Python runner writes, so either
// implementation's runs and baselines can be compared with either tool.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Trones21/noCRUD/go/config"
	"github.com/Trones21/noCRUD/go/utils/perf"
)

func main() {
	runID := flag.String("run", "", "Run id to use (default: latest)")
	threshold := flag.Float64("threshold", 20, "Regression threshold in percent")
	metric := flag.String("metric", "mean", "Which metric drives the comparison and gate: mean, p95 or p99")
	setBaseline := flag.Bool("set-baseline", false, "Promote the selected run to the baseline instead of comparing")
	flag.Parse()

	runnerDir := config.RunnerDir()
	runDir := perf.ResolveRunDir(runnerDir, *runID)
	if runDir == "" {
		fmt.Fprintln(os.Stderr, "No perf runs found. Run flows with:  go run ./cmd/nocrud -crud --perf")
		os.Exit(2)
	}

	if *setBaseline {
		if err := perf.SetBaseline(os.Stdout, runnerDir, runDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}

	regressions, err := perf.CompareAndPrint(os.Stdout, runnerDir, runDir, *threshold, *metric)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(regressions) > 0 {
		os.Exit(1)
	}
}
