package perf_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Trones21/noCRUD/go/utils/perf"
)

// runner sets up an isolated perf root for one test.
func runner(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(perf.EnvEnabled, "1")
	t.Setenv(perf.EnvRunID, "")
	t.Setenv(perf.EnvSHA, "testsha")
	t.Setenv(perf.EnvDir, filepath.Join(dir, "perf"))
	return dir
}

func TestEnabledFollowsTheEnvironment(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{{"", false}, {"0", false}, {"false", false}, {"False", false}, {"1", true}, {"yes", true}} {
		t.Setenv(perf.EnvEnabled, tc.value)
		if got := perf.Enabled(); got != tc.want {
			t.Errorf("Enabled() with %q = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestCollectorWritesOneFilePerFlow(t *testing.T) {
	dir := runner(t)

	run, err := perf.InitRun(dir)
	if err != nil {
		t.Fatalf("InitRun: %v", err)
	}

	actor := run.Collector("actor")
	actor.Record("create_object", "actor", 12500*time.Microsecond)
	actor.Record("get_object_by_id", "actor", 8*time.Millisecond)

	universe := run.Collector("universe")
	universe.Record("create_object", "universe", 20*time.Millisecond)

	for _, c := range []*perf.Collector{actor, universe} {
		if err := c.Flush(); err != nil {
			t.Fatalf("Flush: %v", err)
		}
	}

	// One file per flow is what makes concurrent flows safe to collect.
	files, _ := filepath.Glob(filepath.Join(run.Dir, "*.ndjson"))
	if len(files) != 2 {
		t.Fatalf("wrote %d files, want 2: %v", len(files), files)
	}

	records, err := perf.LoadRecords(run.Dir)
	if err != nil {
		t.Fatalf("LoadRecords: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("loaded %d records, want 3", len(records))
	}
	for _, rec := range records {
		if rec.RunID != run.ID || rec.GitSHA != "testsha" {
			t.Errorf("record not stamped with the run: %+v", rec)
		}
	}
}

// The schema has to stay identical to the Python runner's, or the two can't
// share a baseline.
func TestRecordSchemaMatchesPython(t *testing.T) {
	dir := runner(t)
	run, err := perf.InitRun(dir)
	if err != nil {
		t.Fatal(err)
	}

	c := run.Collector("actor")
	c.Record("create_object", "actor", 12345*time.Microsecond)
	if err := c.Flush(); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(run.Dir, "actor.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(raw), &decoded); err != nil {
		t.Fatalf("record is not valid JSON: %v", err)
	}

	for _, key := range []string{"run_id", "git_sha", "ts", "flow", "op", "endpoint", "ms"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("record is missing the %q key: %s", key, raw)
		}
	}
	if got := decoded["ms"]; got != 12.345 {
		t.Errorf("ms = %v, want 12.345", got)
	}
}

func TestFlowNamesAreMadeFilesystemSafe(t *testing.T) {
	dir := runner(t)
	run, err := perf.InitRun(dir)
	if err != nil {
		t.Fatal(err)
	}

	c := run.Collector("flows/with spaces")
	c.Record("get", "x", time.Millisecond)
	if err := c.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(run.Dir, "flows_with_spaces.ndjson")); err != nil {
		t.Errorf("expected a sanitised filename: %v", err)
	}
}

func TestNilCollectorIsANoOp(t *testing.T) {
	var run *perf.Run
	c := run.Collector("actor")
	c.Record("create_object", "actor", time.Millisecond)
	if err := c.Flush(); err != nil {
		t.Errorf("Flush on a nil collector: %v", err)
	}
}

func TestAggregateAndPercentiles(t *testing.T) {
	endpoint := "actor"
	records := []perf.Record{}
	for _, ms := range []float64{10, 20, 30, 40, 100} {
		records = append(records, perf.Record{Flow: "actor", Op: "get", Endpoint: &endpoint, MS: ms})
	}

	agg := perf.Aggregate(records)
	stats, ok := agg[perf.Key{Flow: "actor", Op: "get", Endpoint: "actor"}]
	if !ok {
		t.Fatalf("no stats for the key: %v", agg)
	}
	if stats.Count != 5 {
		t.Errorf("count = %d, want 5", stats.Count)
	}
	if stats.Mean != 40 {
		t.Errorf("mean = %v, want 40", stats.Mean)
	}
	if stats.Min != 10 || stats.Max != 100 {
		t.Errorf("min/max = %v/%v, want 10/100", stats.Min, stats.Max)
	}
	// Nearest rank: ceil(0.95*5) = 5 → the 5th value.
	if stats.P95 != 100 || stats.P99 != 100 {
		t.Errorf("p95/p99 = %v/%v, want 100/100", stats.P95, stats.P99)
	}
}

func TestSingleSamplePercentileIsThatSample(t *testing.T) {
	endpoint := "actor"
	agg := perf.Aggregate([]perf.Record{{Flow: "actor", Op: "get", Endpoint: &endpoint, MS: 7}})
	stats := agg[perf.Key{Flow: "actor", Op: "get", Endpoint: "actor"}]
	if stats.P95 != 7 || stats.P99 != 7 || stats.Mean != 7 {
		t.Errorf("got %+v, want every metric to be 7", stats)
	}
}

func TestCompareDetectsRegressionsPastTheThreshold(t *testing.T) {
	dir := runner(t)

	baseline := []perf.Record{
		record("actor", "create_object", "actor", 10),
		record("actor", "get_object_by_id", "actor", 10),
	}
	current := []perf.Record{
		record("actor", "create_object", "actor", 30), // +200%
		record("actor", "get_object_by_id", "actor", 11),
		record("actor", "delete_object_by_id", "actor", 5), // new key
	}
	writeNDJSON(t, perf.BaselinePath(dir), baseline)
	runDir := filepath.Join(perf.RunsDir(dir), "20240101T000000_000000Z")
	writeNDJSON(t, filepath.Join(runDir, "actor.ndjson"), current)

	var out bytes.Buffer
	regressions, err := perf.CompareAndPrint(&out, dir, runDir, 20, "mean")
	if err != nil {
		t.Fatalf("CompareAndPrint: %v", err)
	}
	if len(regressions) != 1 {
		t.Fatalf("found %d regressions, want 1: %v", len(regressions), regressions)
	}
	if regressions[0].Key.Op != "create_object" {
		t.Errorf("flagged the wrong key: %v", regressions[0].Key)
	}
	// A key with no baseline is reported as new, never as a regression.
	if !strings.Contains(out.String(), "new") {
		t.Errorf("expected the new key to be marked:\n%s", out.String())
	}
}

func TestCompareWithNoBaselineIsClean(t *testing.T) {
	dir := runner(t)
	runDir := filepath.Join(perf.RunsDir(dir), "20240101T000000_000000Z")
	writeNDJSON(t, filepath.Join(runDir, "actor.ndjson"), []perf.Record{
		record("actor", "create_object", "actor", 10),
	})

	var out bytes.Buffer
	regressions, err := perf.CompareAndPrint(&out, dir, runDir, 20, "mean")
	if err != nil {
		t.Fatalf("CompareAndPrint: %v", err)
	}
	if len(regressions) != 0 {
		t.Errorf("regressions without a baseline: %v", regressions)
	}
	if !strings.Contains(out.String(), "No baseline yet") {
		t.Errorf("expected the missing baseline to be called out:\n%s", out.String())
	}
}

func TestCompareRejectsAnUnknownMetric(t *testing.T) {
	dir := runner(t)
	if _, err := perf.CompareAndPrint(&bytes.Buffer{}, dir, dir, 20, "median"); err == nil {
		t.Error("expected an error for an unsupported metric")
	}
}

func TestSetBaselinePromotesARun(t *testing.T) {
	dir := runner(t)
	runDir := filepath.Join(perf.RunsDir(dir), "20240101T000000_000000Z")
	writeNDJSON(t, filepath.Join(runDir, "actor.ndjson"), []perf.Record{
		record("actor", "create_object", "actor", 10),
		record("actor", "get_object_by_id", "actor", 12),
	})

	if err := perf.SetBaseline(&bytes.Buffer{}, dir, runDir); err != nil {
		t.Fatalf("SetBaseline: %v", err)
	}
	promoted, err := perf.LoadRecords(perf.BaselinePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if len(promoted) != 2 {
		t.Errorf("baseline has %d records, want 2", len(promoted))
	}
}

func TestLatestRunDirPicksTheNewest(t *testing.T) {
	dir := runner(t)
	for _, id := range []string{"20240101T000000_000000Z", "20240301T000000_000000Z", "20240201T000000_000000Z"} {
		if err := os.MkdirAll(filepath.Join(perf.RunsDir(dir), id), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := filepath.Base(perf.LatestRunDir(dir)); got != "20240301T000000_000000Z" {
		t.Errorf("latest = %q", got)
	}
	if got := perf.ResolveRunDir(dir, "nope"); got != "" {
		t.Errorf("ResolveRunDir for a missing run = %q, want empty", got)
	}
}

func TestLoadRecordsToleratesAMissingPath(t *testing.T) {
	records, err := perf.LoadRecords(filepath.Join(t.TempDir(), "nothing-here"))
	if err != nil {
		t.Errorf("LoadRecords on a missing path: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("got %d records", len(records))
	}
}

func record(flow, op, endpoint string, ms float64) perf.Record {
	return perf.Record{Flow: flow, Op: op, Endpoint: &endpoint, MS: ms, RunID: "r", GitSHA: "s"}
}

func writeNDJSON(t *testing.T, path string, records []perf.Record) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			t.Fatal(err)
		}
	}
}
