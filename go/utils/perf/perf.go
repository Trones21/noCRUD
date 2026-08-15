// Package perf persists request timings for regression tracking — the Go
// counterpart of python/utils/perf.py.
//
// Off by default: zero behaviour change unless enabled. Turn it on with the env
// var NOCRUD_PERF=1 (the --perf flag on the runner does this for you). When
// enabled, every timed request is recorded and written to
// perf/runs/<run_id>/<flow>.ndjson, so runs can be compared over time — e.g. in
// CI, before/after a code change, to see how your API's timings moved.
//
// Records are newline-delimited JSON, one object per timed call:
//
//	{"run_id", "git_sha", "ts", "flow", "op", "endpoint", "ms"}
//
// This is byte-for-byte the schema the Python runner writes, and the op names
// match its APIClient method names (create_object, get_object_by_id, …). A run
// captured by either runner can therefore be read, compared and baselined by
// the other — including by python/perf_report.py.
//
// One file per flow (rather than a single shared file) keeps parallel mode
// safe. The Python runner gets that for free by running each flow in its own
// process; here flows are goroutines in one process, so each flow gets its own
// Collector and the writes never overlap.
package perf

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Env vars, shared with the Python implementation so the two runners agree.
const (
	EnvEnabled = "NOCRUD_PERF"
	EnvRunID   = "NOCRUD_PERF_RUN_ID"
	EnvSHA     = "NOCRUD_PERF_SHA"
	EnvDir     = "NOCRUD_PERF_DIR"
)

// Record is one timed request.
type Record struct {
	RunID    string  `json:"run_id"`
	GitSHA   string  `json:"git_sha"`
	TS       string  `json:"ts"`
	Flow     string  `json:"flow"`
	Op       string  `json:"op"`
	Endpoint *string `json:"endpoint"`
	MS       float64 `json:"ms"`
}

// Run is one invocation of the runner. Create it once in main and hand out a
// Collector per flow.
type Run struct {
	ID      string
	GitSHA  string
	Dir     string // perf/runs/<id>
	Enabled bool
}

// Enabled reports whether perf collection is switched on via the environment.
func Enabled() bool {
	switch os.Getenv(EnvEnabled) {
	case "", "0", "false", "False":
		return false
	}
	return true
}

// Root is the perf/ directory: <runner>/perf, or NOCRUD_PERF_DIR.
func Root(runnerDir string) string {
	if d := os.Getenv(EnvDir); d != "" {
		return d
	}
	return filepath.Join(runnerDir, "perf")
}

// RunsDir is perf/runs.
func RunsDir(runnerDir string) string { return filepath.Join(Root(runnerDir), "runs") }

// BaselinePath is perf/baseline.ndjson — the committed reference to compare
// against.
func BaselinePath(runnerDir string) string {
	return filepath.Join(Root(runnerDir), "baseline.ndjson")
}

// InitRun creates a run id, resolves the git sha once, and makes the run
// directory. Call it once, before running any flows.
//
// The run id has microsecond precision so two runs started in the same second
// don't collide, and is fixed width so run ids sort lexicographically by time.
func InitRun(runnerDir string) (*Run, error) {
	now := time.Now().UTC()
	id := os.Getenv(EnvRunID)
	if id == "" {
		id = fmt.Sprintf("%s_%06dZ", now.Format("20060102T150405"), now.Nanosecond()/1000)
	}
	sha := os.Getenv(EnvSHA)
	if sha == "" {
		sha = gitSHA(runnerDir)
	}

	run := &Run{
		ID:      id,
		GitSHA:  sha,
		Dir:     filepath.Join(RunsDir(runnerDir), id),
		Enabled: true,
	}

	// Export both so anything we shell out to shares them.
	_ = os.Setenv(EnvEnabled, "1")
	_ = os.Setenv(EnvRunID, id)
	_ = os.Setenv(EnvSHA, sha)

	if err := os.MkdirAll(run.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("perf: creating run dir: %w", err)
	}
	return run, nil
}

// Collector buffers one flow's timings. A nil *Collector is valid and does
// nothing, so callers never have to nil-check before recording.
type Collector struct {
	run  *Run
	flow string

	mu      sync.Mutex
	records []Record
}

// Collector returns a collector for one flow. A nil *Run yields a nil
// collector, which is a no-op.
func (r *Run) Collector(flow string) *Collector {
	if r == nil || !r.Enabled {
		return nil
	}
	return &Collector{run: r, flow: flow}
}

// Record buffers one timed request. Endpoint may be empty, which is recorded as
// JSON null — matching the Python runner, which records None for calls whose
// first argument isn't a resource name.
func (c *Collector) Record(op, endpoint string, d time.Duration) {
	if c == nil {
		return
	}
	var ep *string
	if endpoint != "" {
		ep = &endpoint
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, Record{
		RunID:    c.run.ID,
		GitSHA:   c.run.GitSHA,
		TS:       time.Now().UTC().Format(time.RFC3339Nano),
		Flow:     c.flow,
		Op:       op,
		Endpoint: ep,
		MS:       round3(float64(d.Nanoseconds()) / 1e6),
	})
}

// Flush writes this flow's buffered records to disk and clears the buffer. Safe
// to call unconditionally at a flow boundary.
func (c *Collector) Flush() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	records := c.records
	c.records = nil
	c.mu.Unlock()

	if len(records) == 0 {
		return nil
	}
	if err := os.MkdirAll(c.run.Dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(c.run.Dir, safeName(c.flow)+".ndjson")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, rec := range records {
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}
	return nil
}

func gitSHA(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func safeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func round3(f float64) float64 {
	return float64(int64(f*1000+0.5)) / 1000
}
