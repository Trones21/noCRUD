// Comparison of persisted timings across runs — the Go counterpart of
// python/perf_report.py. It lives in this package (rather than in the command)
// so the runner can print the same table at the end of a --perf run.

package perf

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Metrics are the statistics a comparison can be gated on.
var Metrics = []string{"mean", "p95", "p99"}

// Key groups samples: one endpoint's one operation within one flow.
type Key struct {
	Flow     string
	Op       string
	Endpoint string
}

func (k Key) String() string { return fmt.Sprintf("%s · %s · %s", k.Flow, k.Op, k.Endpoint) }

// Stats summarises the samples collected for one Key.
type Stats struct {
	Count int
	Mean  float64
	Min   float64
	Max   float64
	P95   float64
	P99   float64
}

// Metric returns the named statistic.
func (s Stats) Metric(name string) float64 {
	switch name {
	case "p95":
		return s.P95
	case "p99":
		return s.P99
	default:
		return s.Mean
	}
}

// LoadRecords reads NDJSON records from a file, or from every *.ndjson in a
// directory. A missing path yields no records rather than an error, matching
// the Python implementation — "no baseline yet" is a normal state.
func LoadRecords(path string) ([]Record, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var files []string
	if info.IsDir() {
		matches, err := filepath.Glob(filepath.Join(path, "*.ndjson"))
		if err != nil {
			return nil, err
		}
		sort.Strings(matches)
		files = matches
	} else {
		files = []string{path}
	}

	var records []Record
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var rec Record
			if err := json.Unmarshal([]byte(line), &rec); err != nil {
				f.Close()
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			records = append(records, rec)
		}
		err = scanner.Err()
		f.Close()
		if err != nil {
			return nil, err
		}
	}
	return records, nil
}

// Aggregate groups samples by (flow, op, endpoint).
func Aggregate(records []Record) map[Key]Stats {
	groups := map[Key][]float64{}
	for _, r := range records {
		endpoint := ""
		if r.Endpoint != nil {
			endpoint = *r.Endpoint
		}
		k := Key{Flow: r.Flow, Op: r.Op, Endpoint: endpoint}
		groups[k] = append(groups[k], r.MS)
	}

	agg := make(map[Key]Stats, len(groups))
	for k, samples := range groups {
		sort.Float64s(samples)
		sum := 0.0
		for _, s := range samples {
			sum += s
		}
		agg[k] = Stats{
			Count: len(samples),
			Mean:  sum / float64(len(samples)),
			Min:   samples[0],
			Max:   samples[len(samples)-1],
			P95:   percentile(samples, 95),
			P99:   percentile(samples, 99),
		}
	}
	return agg
}

// percentile is nearest-rank — robust for any sample count, including n=1.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	switch {
	case n == 0:
		return 0
	case n == 1:
		return sorted[0]
	}
	idx := int(math.Ceil(p/100.0*float64(n))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx > n-1 {
		idx = n - 1
	}
	return sorted[idx]
}

// Regression is one key whose latency moved past the threshold.
type Regression struct {
	Key      Key
	Baseline float64
	Current  float64
	DeltaPct float64
}

// LatestRunDir returns the most recent run directory, or "" if there are none.
func LatestRunDir(runnerDir string) string {
	entries, err := os.ReadDir(RunsDir(runnerDir))
	if err != nil {
		return ""
	}
	latest := ""
	for _, e := range entries {
		if e.IsDir() && e.Name() > latest {
			latest = e.Name()
		}
	}
	if latest == "" {
		return ""
	}
	return filepath.Join(RunsDir(runnerDir), latest)
}

// ResolveRunDir picks the run to report on: the named one, or the latest.
func ResolveRunDir(runnerDir, runID string) string {
	if runID == "" {
		return LatestRunDir(runnerDir)
	}
	dir := filepath.Join(RunsDir(runnerDir), runID)
	if _, err := os.Stat(dir); err != nil {
		return ""
	}
	return dir
}

// CompareAndPrint prints a current-vs-baseline table for metric and returns the
// regressions found.
//
// Regression detection is on the chosen metric (mean/p95/p99). Percentiles only
// become meaningful once a key has many samples; with one sample per key (plain
// CRUD) p95/p99 equal that sample, and a note is printed to that effect.
func CompareAndPrint(w io.Writer, runnerDir, runDir string, thresholdPct float64, metric string) ([]Regression, error) {
	if !validMetric(metric) {
		return nil, fmt.Errorf("metric must be one of %v, got %q", Metrics, metric)
	}

	currentRecords, err := LoadRecords(runDir)
	if err != nil {
		return nil, err
	}
	baselineRecords, err := LoadRecords(BaselinePath(runnerDir))
	if err != nil {
		return nil, err
	}
	current := Aggregate(currentRecords)
	baseline := Aggregate(baselineRecords)

	fmt.Fprintf(w, "\n=== Perf report: %s  (metric: %s) ===\n", filepath.Base(runDir), metric)
	if len(baseline) == 0 {
		fmt.Fprintln(w, "No baseline yet. Set one with:  go run ./cmd/perfreport --set-baseline")
	}
	if len(current) == 0 {
		fmt.Fprintln(w, "No timings in this run. Did you run flows with --perf?")
		return nil, nil
	}

	keys := make([]Key, 0, len(current))
	labelWidth := 0
	for k := range current {
		keys = append(keys, k)
		if n := len([]rune(k.String())); n > labelWidth {
			labelWidth = n
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })

	header := fmt.Sprintf("%s  %3s  %9s  %9s  %8s", pad("key", labelWidth), "n", "base ms", "now ms", "delta")
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, strings.Repeat("-", len(header)))

	var regressions []Regression
	fewSamples := true
	for _, k := range keys {
		cur := current[k]
		if cur.Count >= 20 {
			fewSamples = false
		}
		now := cur.Metric(metric)

		baseStr, deltaStr, flag := "—", "new", ""
		if base, ok := baseline[k]; ok {
			b := base.Metric(metric)
			delta := 0.0
			if b != 0 {
				delta = (now - b) / b * 100
			}
			baseStr = fmt.Sprintf("%.1f", b)
			deltaStr = fmt.Sprintf("%+.1f%%", delta)
			if delta > thresholdPct {
				regressions = append(regressions, Regression{Key: k, Baseline: b, Current: now, DeltaPct: delta})
				flag = "  🔴"
			}
		}
		fmt.Fprintf(w, "%s  %3d  %9s  %9.1f  %8s%s\n", pad(k.String(), labelWidth), cur.Count, baseStr, now, deltaStr, flag)
	}

	if (metric == "p95" || metric == "p99") && fewSamples {
		fmt.Fprintf(w, "\nℹ️  Few samples per key — %s is noisy here. Run a flow many "+
			"times (or use a load flow) for stable percentiles.\n", metric)
	}

	if len(regressions) > 0 {
		fmt.Fprintf(w, "\n🔴 %d regression(s) over %.0f%% (%s):\n", len(regressions), thresholdPct, metric)
		for _, r := range regressions {
			fmt.Fprintf(w, "   %s: %.1f → %.1f ms (%+.1f%%)\n", r.Key, r.Baseline, r.Current, r.DeltaPct)
		}
	} else {
		fmt.Fprintf(w, "\n✅ No %s regressions over %.0f%% threshold.\n", metric, thresholdPct)
	}
	return regressions, nil
}

// SetBaseline promotes a run to the committed baseline.
func SetBaseline(w io.Writer, runnerDir, runDir string) error {
	records, err := LoadRecords(runDir)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		fmt.Fprintf(w, "Nothing to set: no records in %s\n", runDir)
		return nil
	}
	if err := os.MkdirAll(Root(runnerDir), 0o755); err != nil {
		return err
	}
	path := BaselinePath(runnerDir)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "Baseline set from %s (%d records) → %s\n", filepath.Base(runDir), len(records), path)
	return nil
}

func validMetric(m string) bool {
	for _, v := range Metrics {
		if v == m {
			return true
		}
	}
	return false
}

// pad counts runes, not bytes, so the "·" separators in a key label don't throw
// the column alignment off.
func pad(s string, width int) string {
	if n := len([]rune(s)); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}
