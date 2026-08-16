// Package results writes a machine-readable record of what a run did.
//
// The terminal summary is for you; this file is for everything else — a CI step
// that wants to fail on a specific flow, a dashboard, or the anonymized
// community submission the nocrud-share-results skill builds.
//
// The schema is shared with the Python runner, the same way perf records are, so
// either runner's output can be read by the same tooling.
//
// # This file is LOCAL and is not anonymized
//
// It deliberately contains flow names, which are usually your endpoint names and
// therefore say something about your domain model. That is what makes it useful
// locally, and it is exactly why it is not the thing you upload anywhere. The
// share skill builds a separate payload from a strict field allowlist and shows
// it to you before anything leaves your machine.
package results

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Schema identifies the format. Bump it when the shape changes so consumers can
// tell which one they're holding.
const Schema = "nocrud.results/v1"

// File is one run.
type File struct {
	Schema string `json:"schema"`
	// Runner is "go" or "python".
	Runner string `json:"runner"`
	// RunnerSHA is the noCRUD commit the runner was built from, when known.
	// This is noCRUD's sha, not the app-under-test's.
	RunnerSHA string `json:"runner_sha,omitempty"`

	StartedAt string  `json:"started_at"`
	WallMS    float64 `json:"wall_ms"`

	// Mode is "parallel" or "serial"; Jobs is the concurrency cap actually used.
	Mode string `json:"mode"`
	Jobs int    `json:"jobs,omitempty"`

	Passed bool   `json:"passed"`
	Flows  []Flow `json:"flows"`
}

// Flow is one flow's outcome.
type Flow struct {
	Name string  `json:"name"`
	Kind string  `json:"kind"`
	OK   bool    `json:"ok"`
	MS   float64 `json:"ms"`
	// Summary is the same line the terminal printed, with colour stripped.
	Summary string `json:"summary"`
	// Error is the failure message, when there was one.
	Error string `json:"error,omitempty"`
}

// Counts summarises the run. The share skill reports these rather than the
// per-flow detail.
func (f File) Counts() (total, passed, failed, crud, request int) {
	for _, fl := range f.Flows {
		total++
		if fl.OK {
			passed++
		} else {
			failed++
		}
		switch fl.Kind {
		case "crud":
			crud++
		case "request":
			request++
		}
	}
	return
}

// Write saves the file, creating the parent directory if needed.
func Write(path string, f File) error {
	f.Schema = Schema
	if f.StartedAt == "" {
		f.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("results: creating %s: %w", dir, err)
		}
	}

	encoded, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("results: encoding: %w", err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		return fmt.Errorf("results: writing %s: %w", path, err)
	}
	return nil
}

// ansi matches the colour escapes the summary formatter emits. They belong on a
// terminal, not in a JSON string.
var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripANSI removes terminal colour codes.
func StripANSI(s string) string { return ansi.ReplaceAllString(s, "") }
