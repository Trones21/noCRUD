// Package runners executes flows and reports what happened — the Go
// counterpart of python/runners/.
//
// Two modes, same as the Python runner:
//
//   - Serial prints as it goes, against a backend you started yourself.
//   - Parallel gives every flow its own backend and database, and buffers each
//     flow's output so the logs don't interleave into mush.
//
// The difference from Python is what "parallel" costs. There, each flow is an
// OS process (multiprocessing.Pool). Here they're goroutines, so the runner
// itself is nearly free and the concurrency limit exists to protect the
// *backend and database*, not the runner — see Options.Jobs.
package runners

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/apiclient"
	"github.com/Trones21/noCRUD/go/utils/crud"
	"github.com/Trones21/noCRUD/go/utils/perf"
	"github.com/Trones21/noCRUD/go/utils/printing"
	"github.com/Trones21/noCRUD/go/utils/provisioning"
)

// ProvisionFunc creates a flow's isolated environment.
type ProvisionFunc func(ctx context.Context, flowName string, out io.Writer) (*nocrud.Env, error)

// CleanupFunc tears one down.
type CleanupFunc func(ctx context.Context, env *nocrud.Env, out io.Writer)

// Options configure a run.
type Options struct {
	// Parallel gives each flow its own provisioned environment. Default true,
	// matching the Python runner.
	Parallel bool

	// Jobs caps how many flows run at once in parallel mode. Zero means
	// GOMAXPROCS.
	//
	// The cap is about the far end of the connection: each concurrent flow
	// means another backend process and another database, so on a small box
	// the bottleneck is provisioning, not the runner.
	Jobs int

	// Run collects timings when non-nil.
	Run *perf.Run

	// Out is where the runner's own output goes (progress in serial mode,
	// buffered flow output and the summary in parallel mode).
	Out io.Writer

	// Provision and Cleanup default to the ones in utils/provisioning. Swap
	// them to target a backend that isn't Django — or, in a test, to skip
	// provisioning entirely.
	Provision ProvisionFunc
	Cleanup   CleanupFunc
}

func (o Options) provision() ProvisionFunc {
	if o.Provision != nil {
		return o.Provision
	}
	return provisioning.ProvisionEnvForFlow
}

func (o Options) cleanup() CleanupFunc {
	if o.Cleanup != nil {
		return o.Cleanup
	}
	return provisioning.CleanupEnv
}

func (o Options) out() io.Writer {
	if o.Out != nil {
		return o.Out
	}
	return os.Stdout
}

func (o Options) jobs() int {
	if o.Jobs > 0 {
		return o.Jobs
	}
	return runtime.GOMAXPROCS(0)
}

// Result is one flow's outcome.
type Result struct {
	Flow      nocrud.Flow
	Value     any
	Formatted string
	Err       error
	// Duration is how long the flow itself took — not counting provisioning,
	// which is the environment's cost rather than the flow's.
	Duration time.Duration
	// Output is the flow's captured log, in parallel mode. Empty in serial
	// mode, where it went straight to the terminal.
	Output string
}

// Failed reports whether the flow errored.
func (r Result) Failed() bool { return r.Err != nil }

// Run executes flows and returns their results in the order given.
func Run(ctx context.Context, flows []nocrud.Flow, opts Options) []Result {
	if len(flows) == 0 {
		fmt.Fprintln(opts.out(), "No flows to run.")
		return nil
	}
	if opts.Parallel {
		return runParallel(ctx, flows, opts)
	}
	return runSerial(ctx, flows, opts)
}

// ============================================================================
//  Serial
// ============================================================================

// runSerial executes each flow in real time, with no output buffering, against
// the app you already have running.
func runSerial(ctx context.Context, flows []nocrud.Flow, opts Options) []Result {
	out := opts.out()
	results := make([]Result, 0, len(flows))

	for _, flow := range flows {
		env := nocrud.EnvFromProcess()
		res := execFlow(ctx, flow, env, out, opts.Run)
		if res.Err == nil {
			fmt.Fprintf(out, "\n%s\n\n", res.Formatted)
		}
		results = append(results, res)
	}
	return results
}

// ============================================================================
//  Parallel
// ============================================================================

// runParallel gives each flow its own environment and captures its output, so
// the logs stay readable even though everything is happening at once.
func runParallel(ctx context.Context, flows []nocrud.Flow, opts Options) []Result {
	out := opts.out()
	fmt.Fprintf(out, "Parallel Run Begin (%d at a time)\n\n", opts.jobs())

	results := make([]Result, len(flows))
	sem := make(chan struct{}, opts.jobs())
	var wg sync.WaitGroup

	for i, flow := range flows {
		wg.Add(1)
		go func(i int, flow nocrud.Flow) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = runIsolatedFlow(ctx, flow, opts)
		}(i, flow)
	}
	wg.Wait()

	for _, res := range results {
		fmt.Fprint(out, res.Output)
	}
	return results
}

// runIsolatedFlow provisions an environment, runs the flow into a buffer, and
// tears the environment down again.
//
// A failing flow keeps its database: the run is over, but the state that broke
// it is still there to inspect. Clean them up with
// python/drop_dbs_by_pattern.sh when you're done.
func runIsolatedFlow(ctx context.Context, flow nocrud.Flow, opts Options) Result {
	var buf bytes.Buffer

	start := time.Now()
	env, err := opts.provision()(ctx, flow.Name, &buf)
	fmt.Fprintf(&buf, "[⏱] %s: %.4fs\n", printing.PadRight("provision env", 30), time.Since(start).Seconds())

	if err != nil {
		if env != nil {
			env.PersistDB = true
			opts.cleanup()(ctx, env, &buf)
		}
		fmt.Fprintf(&buf, "\nFlow '%s' failed to provision: %v\n", flow.Name, err)
		return Result{Flow: flow, Err: err, Formatted: "Fail: " + err.Error(), Output: buf.String()}
	}

	res := execFlow(ctx, flow, env, &buf, opts.Run)
	if res.Err != nil {
		env.PersistDB = true
	} else {
		fmt.Fprintf(&buf, "\n\n%s\n\n", res.Formatted)
	}

	opts.cleanup()(ctx, env, &buf)
	res.Output = buf.String()
	return res
}

// ============================================================================
//  Shared execution
// ============================================================================

// execFlow runs one flow to completion, whatever it does — including panicking.
//
// A Python flow signals failure by letting an exception escape, and the runner
// turns that into a traceback plus a red mark in the summary. nocrud.Must gives
// Go flows the same option, so a panic is a first-class outcome here rather
// than a crash.
func execFlow(ctx context.Context, flow nocrud.Flow, env *nocrud.Env, out io.Writer, run *perf.Run) (result Result) {
	result = Result{Flow: flow}

	// Registered first, so it runs last — after the recover below has turned a
	// panic into a result. A flow that panicked still took time worth recording.
	start := time.Now()
	defer func() { result.Duration = time.Since(start) }()

	collector := run.Collector(flow.Name)
	defer func() {
		if err := collector.Flush(); err != nil {
			fmt.Fprintf(out, "⚠️ Could not write timings for %s: %v\n", flow.Name, err)
		}
	}()

	c := nocrud.NewCtx(ctx, flow.Name, env, out, collector)
	defer c.Close()

	defer func() {
		if r := recover(); r != nil {
			result.Err = asError(r)
			result.Formatted = "Fail: " + result.Err.Error()
			reportFailure(out, flow.Name, result.Err, debug.Stack())
		}
	}()

	fmt.Fprintf(out, "Running flow: %s\n%s\n", flow.Name, strings.Repeat("-", 60))

	value, err := flow.Fn(c)
	result.Value = value
	result.Err = err
	result.Formatted = format(value, err)
	if err != nil {
		reportFailure(out, flow.Name, err, nil)
	}
	return result
}

// asError turns a recovered panic into an error, unwrapping the ones nocrud
// raised on purpose so they read as plain failures rather than crashes.
func asError(r any) error {
	switch v := r.(type) {
	case *nocrud.FlowPanic:
		return v.Err
	case error:
		return fmt.Errorf("panic: %w", v)
	default:
		return fmt.Errorf("panic: %v", r)
	}
}

// reportFailure explains what went wrong, with a stack trace when there is one
// worth reading. A backend that isn't running is the exception: that needs a
// sentence, not a stack.
func reportFailure(out io.Writer, flowName string, err error, stack []byte) {
	if apiclient.IsConnectionError(err) {
		fmt.Fprintf(out, "\nFlow '%s' failed: Unable to connect to the backend. Is it running?\n", flowName)
		return
	}
	fmt.Fprintf(out, "\nFlow '%s' failed with error: %v\n", flowName, err)
	if len(stack) > 0 {
		fmt.Fprintf(out, "Stack trace:\n%s\n", stack)
	}
}

// format renders a flow's return value for the summary.
func format(value any, err error) string {
	if err != nil {
		return "Fail: " + err.Error()
	}
	switch v := value.(type) {
	case crud.Result:
		return printing.FormatCRUD(v)
	case map[string]bool:
		return printing.FormatCRUD(v)
	case nil:
		return "ok"
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

// ============================================================================
//  Summary
// ============================================================================

// PrintSummary prints the results table and reports whether everything passed.
func PrintSummary(out io.Writer, results []Result) bool {
	printing.GroupSeparator(out, "Results Summary")

	width := 0
	for _, r := range results {
		if n := len([]rune(r.Flow.Name)); n > width {
			width = n
		}
	}

	passed := true
	for _, r := range results {
		if r.Failed() {
			passed = false
		}
		fmt.Fprintf(out, "%s: %s\n", printing.PadRight(r.Flow.Name, width), r.Formatted)
	}
	return passed
}
