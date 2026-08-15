package runners_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/runners"
	"github.com/Trones21/noCRUD/go/utils/crud"
)

// stubbed replaces provisioning, so the runner's own behaviour can be tested
// without a Django app or a database anywhere in sight.
func stubbed(out io.Writer) runners.Options {
	return runners.Options{
		Out: out,
		Provision: func(ctx context.Context, flowName string, w io.Writer) (*nocrud.Env, error) {
			fmt.Fprintf(w, "provisioned %s\n", flowName)
			return &nocrud.Env{DBName: "db_" + flowName, AppPort: 8000}, nil
		},
		Cleanup: func(ctx context.Context, env *nocrud.Env, w io.Writer) {
			fmt.Fprintf(w, "cleaned up %s (persist=%v)\n", env.DBName, env.PersistDB)
		},
	}
}

func flow(name string, fn nocrud.Func) nocrud.Flow {
	return nocrud.Flow{Name: name, Kind: nocrud.Request, Fn: fn}
}

func TestSerialRunsInOrderAndPrintsAsItGoes(t *testing.T) {
	var order []string
	var out bytes.Buffer

	step := func(name string) nocrud.Func {
		return func(c *nocrud.Ctx) (any, error) {
			order = append(order, name)
			c.Printf("inside %s\n", name)
			return name + " done", nil
		}
	}

	opts := stubbed(&out)
	opts.Parallel = false
	results := runners.Run(context.Background(), []nocrud.Flow{
		flow("first", step("first")),
		flow("second", step("second")),
	}, opts)

	if strings.Join(order, ",") != "first,second" {
		t.Errorf("ran in order %v, want first,second", order)
	}
	if len(results) != 2 || results[0].Formatted != "first done" {
		t.Errorf("results = %+v", results)
	}
	// Serial mode prints live rather than buffering.
	if !strings.Contains(out.String(), "inside first") {
		t.Errorf("flow output did not reach the terminal:\n%s", out.String())
	}
	if results[0].Output != "" {
		t.Errorf("serial mode should not buffer, got %q", results[0].Output)
	}
}

func TestParallelKeepsEachFlowsOutputSeparate(t *testing.T) {
	var out bytes.Buffer

	noisy := func(name string) nocrud.Func {
		return func(c *nocrud.Ctx) (any, error) {
			for i := 0; i < 20; i++ {
				c.Printf("%s line %d\n", name, i)
			}
			return name, nil
		}
	}

	opts := stubbed(&out)
	opts.Parallel = true
	results := runners.Run(context.Background(), []nocrud.Flow{
		flow("alpha", noisy("alpha")),
		flow("beta", noisy("beta")),
		flow("gamma", noisy("gamma")),
	}, opts)

	// The whole point of buffering: one flow's lines never land inside
	// another's.
	for _, res := range results {
		for _, other := range []string{"alpha", "beta", "gamma"} {
			if other == res.Flow.Name {
				continue
			}
			if strings.Contains(res.Output, other+" line") {
				t.Errorf("%s's buffer contains %s's output", res.Flow.Name, other)
			}
		}
	}
	// Results come back in the order asked for, however they interleaved.
	for i, want := range []string{"alpha", "beta", "gamma"} {
		if results[i].Flow.Name != want {
			t.Errorf("results[%d] = %s, want %s", i, results[i].Flow.Name, want)
		}
	}
}

func TestParallelGivesEachFlowItsOwnEnvironment(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]string{}

	opts := stubbed(io.Discard)
	opts.Parallel = true

	capture := func(c *nocrud.Ctx) (any, error) {
		mu.Lock()
		defer mu.Unlock()
		seen[c.FlowName] = c.Env.DBName
		return nil, nil
	}

	runners.Run(context.Background(), []nocrud.Flow{
		flow("one", capture),
		flow("two", capture),
	}, opts)

	if seen["one"] == seen["two"] {
		t.Errorf("both flows saw the same database: %v", seen)
	}
}

func TestFailingFlowKeepsItsDatabase(t *testing.T) {
	var out bytes.Buffer
	opts := stubbed(&out)
	opts.Parallel = true

	results := runners.Run(context.Background(), []nocrud.Flow{
		flow("ok", func(c *nocrud.Ctx) (any, error) { return "fine", nil }),
		flow("broken", func(c *nocrud.Ctx) (any, error) { return nil, errors.New("boom") }),
	}, opts)

	if !strings.Contains(results[0].Output, "persist=false") {
		t.Errorf("a passing flow should drop its database:\n%s", results[0].Output)
	}
	// A failed flow's database is left behind so there is something to inspect.
	if !strings.Contains(results[1].Output, "persist=true") {
		t.Errorf("a failing flow should keep its database:\n%s", results[1].Output)
	}
}

func TestPanicBecomesAFailedFlowNotACrash(t *testing.T) {
	opts := stubbed(io.Discard)
	opts.Parallel = false

	results := runners.Run(context.Background(), []nocrud.Flow{
		flow("must", func(c *nocrud.Ctx) (any, error) {
			// What nocrud.Must does when a step fails.
			return nocrud.Must("", errors.New("400 Client Error")), nil
		}),
		flow("nil-deref", func(c *nocrud.Ctx) (any, error) {
			var m map[string]string
			m["boom"] = "x" // assignment to entry in nil map
			return nil, nil
		}),
		flow("after", func(c *nocrud.Ctx) (any, error) { return "still running", nil }),
	}, opts)

	if results[0].Err == nil || !strings.Contains(results[0].Formatted, "400 Client Error") {
		t.Errorf("Must panic was not reported as a plain failure: %+v", results[0])
	}
	if strings.Contains(results[0].Formatted, "panic") {
		t.Errorf("a deliberate Must failure should not read as a panic: %q", results[0].Formatted)
	}
	if results[1].Err == nil || !strings.Contains(results[1].Formatted, "panic") {
		t.Errorf("an accidental panic should be reported as one: %+v", results[1])
	}
	// And the run carries on.
	if results[2].Err != nil {
		t.Errorf("a panicking flow took the run down with it: %+v", results[2])
	}
}

func TestProvisionFailureIsAttributedToItsFlow(t *testing.T) {
	opts := stubbed(io.Discard)
	opts.Parallel = true
	opts.Provision = func(ctx context.Context, flowName string, w io.Writer) (*nocrud.Env, error) {
		if flowName == "unprovisionable" {
			return &nocrud.Env{DBName: "half-made"}, errors.New("could not create database")
		}
		return &nocrud.Env{DBName: "db_" + flowName, AppPort: 8000}, nil
	}

	results := runners.Run(context.Background(), []nocrud.Flow{
		flow("unprovisionable", func(c *nocrud.Ctx) (any, error) { return "should not run", nil }),
		flow("fine", func(c *nocrud.Ctx) (any, error) { return "ran", nil }),
	}, opts)

	if results[0].Err == nil {
		t.Error("expected the provisioning failure to be reported")
	}
	if !strings.Contains(results[0].Output, "cleaned up half-made (persist=true)") {
		t.Errorf("a half-provisioned env should still be cleaned up:\n%s", results[0].Output)
	}
	if results[1].Err != nil {
		t.Errorf("the other flow should be unaffected: %v", results[1].Err)
	}
}

func TestCRUDResultsAreFormattedAsTicks(t *testing.T) {
	var out bytes.Buffer
	opts := stubbed(&out)
	opts.Parallel = false

	results := runners.Run(context.Background(), []nocrud.Flow{
		{Name: "actor", Kind: nocrud.CRUD, Fn: func(c *nocrud.Ctx) (any, error) {
			return crud.Result{"create": true, "read": true, "update": false, "delete": true}, nil
		}},
	}, opts)

	if !strings.HasPrefix(results[0].Formatted, "C:") || !strings.Contains(results[0].Formatted, "U:") {
		t.Errorf("crud result was not formatted: %q", results[0].Formatted)
	}
}

func TestPrintSummaryReportsOverallOutcome(t *testing.T) {
	var out bytes.Buffer
	opts := stubbed(io.Discard)
	opts.Parallel = false

	results := runners.Run(context.Background(), []nocrud.Flow{
		flow("passing", func(c *nocrud.Ctx) (any, error) { return "ok", nil }),
	}, opts)
	if !runners.PrintSummary(&out, results) {
		t.Error("PrintSummary reported a failure for a passing run")
	}
	if !strings.Contains(out.String(), "Results Summary") {
		t.Errorf("summary banner missing:\n%s", out.String())
	}

	results = runners.Run(context.Background(), []nocrud.Flow{
		flow("failing", func(c *nocrud.Ctx) (any, error) { return nil, errors.New("nope") }),
	}, opts)
	if runners.PrintSummary(io.Discard, results) {
		t.Error("PrintSummary reported success for a failing run")
	}
}

func TestJobsLimitsConcurrency(t *testing.T) {
	var mu sync.Mutex
	running, peak := 0, 0

	opts := stubbed(io.Discard)
	opts.Parallel = true
	opts.Jobs = 2

	var flows []nocrud.Flow
	for i := 0; i < 8; i++ {
		flows = append(flows, flow(fmt.Sprintf("flow%d", i), func(c *nocrud.Ctx) (any, error) {
			mu.Lock()
			running++
			if running > peak {
				peak = running
			}
			mu.Unlock()

			// Long enough that a broken limiter would overlap.
			for i := 0; i < 1000; i++ {
				c.Printf("")
			}

			mu.Lock()
			running--
			mu.Unlock()
			return nil, nil
		}))
	}

	runners.Run(context.Background(), flows, opts)
	if peak > 2 {
		t.Errorf("peak concurrency = %d, want at most 2", peak)
	}
}
