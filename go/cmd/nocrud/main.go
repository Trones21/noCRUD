// Command nocrud runs backend user flow simulations — the Go counterpart of
// python/noCRUD.py.
//
// Architecture of any test runner (this file is the whole of steps 1 and 3):
//
//  1. Get the set of tests to be run
//  2. Run tests and collect results
//  3. Print results
//
// Steps 2 and 3 aren't completely mutually exclusive — serial mode prints a few
// specific things while running and then just prints a summary at the end.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Trones21/noCRUD/go/config"
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/runners"
	"github.com/Trones21/noCRUD/go/utils/dbclient"
	"github.com/Trones21/noCRUD/go/utils/perf"
	"github.com/Trones21/noCRUD/go/utils/printing"

	// Registers the flows. Every file in this package that calls
	// nocrud.Register in its init() shows up here — the Go equivalent of the
	// Python runner's folder collector.
	_ "github.com/Trones21/noCRUD/go/flows"
)

type options struct {
	serial   bool
	perf     bool
	jobs     int
	flows    stringList
	crud     bool
	request  bool
	all      bool
	list     bool
	metric   string
	threshld float64
}

func main() {
	os.Exit(run())
}

func run() int {
	opts := parseFlags()

	// ---------------------------------------------------------------
	// Options that don't run any flows
	// ---------------------------------------------------------------
	if opts.list {
		listFlows(os.Stdout)
		return 0
	}

	selected, err := selectFlows(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	// ---------------------------------------------------------------
	// "Main"
	// ---------------------------------------------------------------
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var run *perf.Run
	if opts.perf {
		run, err = perf.InitRun(config.RunnerDir())
		if err != nil {
			fmt.Fprintln(os.Stderr, "perf:", err)
			return 2
		}
		fmt.Printf("⏱  Perf collection on — run id %s\n", run.ID)
	}

	start := time.Now()

	parallel := !opts.serial
	if opts.serial {
		// Serial only. Provisioning takes care of this per flow in parallel
		// mode, where each flow has its own database to begin with.
		printing.GroupSeparator(os.Stdout, "Initial Setup")
		if err := resetSerialDatabase(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}

	printing.GroupSeparator(os.Stdout, "Run Flows")
	fmt.Printf("Flows to run: %v\n", names(selected))

	results := runners.Run(ctx, selected, runners.Options{
		Parallel: parallel,
		Jobs:     opts.jobs,
		Run:      run,
		Out:      os.Stdout,
	})

	passed := runners.PrintSummary(os.Stdout, results)

	printing.Rule(os.Stdout, 80)
	fmt.Printf("\nTest runner took: %.6f seconds\n", time.Since(start).Seconds())

	if run != nil {
		printing.GroupSeparator(os.Stdout, "Timings")
		if _, err := perf.CompareAndPrint(os.Stdout, config.RunnerDir(), run.Dir, opts.threshld, opts.metric); err != nil {
			fmt.Fprintln(os.Stderr, "perf:", err)
		}
		fmt.Println("\nSet this run as the baseline to compare against next time:\n" +
			"  go run ./cmd/perfreport --set-baseline")
	}

	if !passed {
		return 1
	}
	return 0
}

// selectFlows resolves the mutually exclusive selection flags into the flows to
// run.
func selectFlows(opts options) ([]nocrud.Flow, error) {
	chosen := 0
	for _, set := range []bool{len(opts.flows) > 0, opts.crud, opts.request, opts.all} {
		if set {
			chosen++
		}
	}
	switch {
	case chosen == 0:
		return nil, fmt.Errorf("one of -f/--flows, -crud, -req/--request_flows, -coll/--collected or -l/--list is required\n\nRegistered flows:\n%s", flowListing())
	case chosen > 1:
		return nil, fmt.Errorf("-f/--flows, -crud, -req/--request_flows and -coll/--collected are mutually exclusive")
	}

	switch {
	case len(opts.flows) > 0:
		return nocrud.Select(opts.flows)
	case opts.crud:
		return requireNonEmpty(nocrud.ByKind(nocrud.CRUD), "crud")
	case opts.request:
		return requireNonEmpty(nocrud.ByKind(nocrud.Request), "request")
	default:
		return requireNonEmpty(nocrud.All(), "registered")
	}
}

func requireNonEmpty(flows []nocrud.Flow, kind string) ([]nocrud.Flow, error) {
	if len(flows) == 0 {
		return nil, fmt.Errorf("no %s flows are registered — add one to the flows package (see go/flows/Readme.md)", kind)
	}
	return flows, nil
}

// resetSerialDatabase drops, migrates and reloads the database the app you
// started is already using.
func resetSerialDatabase(ctx context.Context) error {
	env := nocrud.EnvFromProcess()
	db, err := dbclient.New(ctx, env.DBConfig, os.Stdout)
	if err != nil {
		return fmt.Errorf("serial setup: %w", err)
	}
	defer db.Close(ctx)
	return db.Reset(ctx, env.Environ())
}

func listFlows(w io.Writer) {
	fmt.Fprint(w, flowListing())
}

func flowListing() string {
	var b strings.Builder
	writeGroup := func(title string, flows []nocrud.Flow) {
		fmt.Fprintf(&b, "\n%s:\n", title)
		if len(flows) == 0 {
			fmt.Fprintln(&b, "  (none)")
			return
		}
		for _, f := range flows {
			if f.Doc != "" {
				fmt.Fprintf(&b, "  %s — %s\n", f.Name, f.Doc)
			} else {
				fmt.Fprintf(&b, "  %s\n", f.Name)
			}
		}
	}
	writeGroup("CRUD FLOWS", nocrud.ByKind(nocrud.CRUD))
	writeGroup("REQUEST FLOWS", nocrud.ByKind(nocrud.Request))
	return b.String()
}

func names(flows []nocrud.Flow) []string {
	out := make([]string, len(flows))
	for i, f := range flows {
		out[i] = f.Name
	}
	return out
}

// ============================================================================
//  Flags
// ============================================================================

func parseFlags() options {
	var opts options

	// Go's flag package has no concept of long vs short names, so each flag is
	// registered under both spellings the Python CLI accepts.
	boolVar := func(p *bool, usage string, names ...string) {
		for _, n := range names {
			flag.BoolVar(p, n, false, usage)
		}
	}

	boolVar(&opts.serial, "Run flows serially, against an app you started yourself", "s", "serial")
	boolVar(&opts.perf, "Persist request timings for this run and compare against the baseline", "perf")
	boolVar(&opts.crud, "Run all crud flows", "crud")
	boolVar(&opts.request, "Run all request flows", "req", "request_flows")
	boolVar(&opts.all, "Run every registered flow", "coll", "collected")
	boolVar(&opts.list, "List all flows", "l", "list")
	flag.Var(&opts.flows, "f", "Name(s) of the flow(s) to run — repeat the flag or comma-separate")
	flag.Var(&opts.flows, "flows", "Name(s) of the flow(s) to run — repeat the flag or comma-separate")
	flag.IntVar(&opts.jobs, "j", 0, "Max flows running at once in parallel mode (default: GOMAXPROCS)")
	flag.IntVar(&opts.jobs, "jobs", 0, "Max flows running at once in parallel mode (default: GOMAXPROCS)")
	flag.Float64Var(&opts.threshld, "threshold", 20, "Perf regression threshold, in percent")
	flag.StringVar(&opts.metric, "metric", "mean", "Perf metric to compare: mean, p95 or p99")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Run backend user flow simulations.\n\nUsage:\n  %s [flags]\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprint(flag.CommandLine.Output(), flowListing())
	}
	flag.Parse()

	return opts
}

// stringList collects a repeated (or comma-separated) flag value, standing in
// for argparse's nargs="+".
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			*s = append(*s, part)
		}
	}
	return nil
}
