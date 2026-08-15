// Package fixtures reads the app's version-controlled test data — the Go
// counterpart of python/utils/fixtures.py.
//
// Fixtures are read fresh on every call rather than cached, so a flow that
// mutates the object it got back (the usual case: tweak one field, then POST
// it) can't leak that change into the next flow.
package fixtures

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Trones21/noCRUD/go/config"
)

// Entry is one record in a Django fixture file.
//
// UnhashedPass is noCRUD's own convention, not Django's: Django expects the
// password in a fixture to be pre-hashed, so a fixture that also wants to be
// loginable carries the plaintext alongside it at the top level. See the
// "A Note on Django Fixtures & Passwords" section in python/docs/README.md.
type Entry struct {
	Model        string         `json:"model"`
	PK           any            `json:"pk"`
	Fields       map[string]any `json:"fields"`
	UnhashedPass string         `json:"unhashed_pass,omitempty"`
}

// Path returns the full path of a fixture file.
func Path(filename string) string {
	return filepath.Join(config.FixturesPath(), filename)
}

// Get opens a fixture file and returns its entries.
func Get(filename string) ([]Entry, error) {
	path := Path(filename)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("fixture file not found: %s", path)
		}
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return entries, nil
}

// GetByIndex returns one entry's fields, ready to POST.
//
// When the entry carries an unhashed_pass it is folded into the returned map
// under that key, so a caller can both create the object and log in as it
// without opening the file twice.
func GetByIndex(filename string, index int) (map[string]any, error) {
	entries, err := Get(filename)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(entries) {
		return nil, fmt.Errorf("fixture %s: index %d out of range (%d entries)", filename, index, len(entries))
	}
	entry := entries[index]
	fields := entry.Fields
	if fields == nil {
		fields = map[string]any{}
	}
	if entry.UnhashedPass != "" {
		fields["unhashed_pass"] = entry.UnhashedPass
	}
	return fields, nil
}

// Credentials returns the username and plaintext password of a user fixture,
// which is what a flow actually wants when it just needs to log in.
func Credentials(filename string, index int) (username, password string, err error) {
	fields, err := GetByIndex(filename, index)
	if err != nil {
		return "", "", err
	}
	username, _ = fields["username"].(string)
	password, _ = fields["unhashed_pass"].(string)
	if username == "" {
		return "", "", fmt.Errorf("fixture %s[%d] has no username field", filename, index)
	}
	if password == "" {
		return "", "", fmt.Errorf(
			"fixture %s[%d] has no unhashed_pass — Django stores passwords hashed, so a "+
				"loginable user fixture must carry the plaintext at the top level "+
				"(see python/docs/README.md)", filename, index)
	}
	return username, password, nil
}

// Add loads fixtures into the database with manage.py loaddata. Passing no
// names loads the minimal set ("users"), matching the Python default.
//
// env is the environment for the subprocess — in parallel mode that is the
// per-flow environment, which is why it is passed in rather than inherited.
func Add(w io.Writer, env []string, names ...string) error {
	if len(names) == 0 {
		names = []string{"users"}
	}
	fmt.Fprintln(w, "adding fixtures", names)
	for _, name := range names {
		file := filepath.Join(config.FixturesPath(), name+".json")
		cmd := exec.Command("python", "manage.py", "loaddata", file)
		cmd.Dir = config.AppDir()
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Parity with the Python runner: a fixture that won't load is
			// reported, not fatal. The flow itself will fail soon enough, and
			// with a better message, if it actually needed the data.
			fmt.Fprintf(w, "Error Adding Fixture: %s: %v\n%s\n", name, err, out)
		}
	}
	return nil
}
