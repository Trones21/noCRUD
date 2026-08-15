// Package config is the one file you edit when you drop this runner next to
// your own app — the Go counterpart of python/config.py.
//
// Everything is resolved once, at first use, and can be overridden by an
// environment variable so the same binary works from a shell, from CI, and from
// an editor's run configuration without edits.
package config

import (
	"os"
	"path/filepath"
	"sync"
)

// Env var overrides. These take precedence over the defaults below.
const (
	EnvAppDir       = "NOCRUD_APP_DIR"       // root of the app under test
	EnvFixturesPath = "NOCRUD_FIXTURES_PATH" // directory holding the fixture json
	EnvRunnerDir    = "NOCRUD_RUNNER_DIR"    // root of this runner (perf/, flows/)
	EnvSettingsPath = "NOCRUD_SETTINGS_PATH" // django settings.py, for the DB match check
	EnvProvision    = "NOCRUD_PROVISION"     // migrate | sql | template
)

// Only the filesystem walk is cached. Everything else is resolved per call, so
// changing an override at runtime takes effect rather than being shadowed by a
// value captured on first use.
var (
	once       sync.Once
	discovered string
)

// AppDir is the root of the application under test. No trailing slash.
func AppDir() string {
	if dir := os.Getenv(EnvAppDir); dir != "" {
		return dir
	}
	// Default mirrors python/config.py: the app sits next to the runner.
	return filepath.Join(filepath.Dir(RunnerDir()), "example_app")
}

// FixturesPath is the directory holding the app's fixture json files.
func FixturesPath() string {
	if dir := os.Getenv(EnvFixturesPath); dir != "" {
		return dir
	}
	return filepath.Join(AppDir(), "api", "fixtures")
}

// RunnerDir is the root of this runner — where perf/ is written.
func RunnerDir() string {
	if dir := os.Getenv(EnvRunnerDir); dir != "" {
		return dir
	}
	once.Do(func() { discovered = findRunnerDir() })
	return discovered
}

// SettingsPath is the Django settings module used by the DB match check.
func SettingsPath() string {
	if p := os.Getenv(EnvSettingsPath); p != "" {
		return p
	}
	return filepath.Join(AppDir(), filepath.Base(AppDir()), "settings.py")
}

// findRunnerDir walks up from the working directory looking for the runner's
// go.mod. A compiled binary can be invoked from anywhere, so unlike Python —
// which can lean on __file__ — the location has to be discovered (or set with
// NOCRUD_RUNNER_DIR).
func findRunnerDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// No go.mod anywhere above us — fall back to the working directory.
			wd, _ := os.Getwd()
			return wd
		}
		dir = parent
	}
}
