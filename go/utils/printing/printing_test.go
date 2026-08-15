package printing_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Trones21/noCRUD/go/utils/printing"
)

func TestFormatCRUDMarksEachOperation(t *testing.T) {
	got := printing.FormatCRUD(map[string]bool{
		"create": true, "read": true, "update": false, "delete": true,
	})

	if !strings.HasPrefix(got, "C:") {
		t.Errorf("format = %q, want it to start with C:", got)
	}
	for _, label := range []string{"C:", "R:", "U:", "D:"} {
		if !strings.Contains(got, label) {
			t.Errorf("format = %q, missing %s", got, label)
		}
	}
	if strings.Count(got, "✔") != 3 || strings.Count(got, "✘") != 1 {
		t.Errorf("format = %q, want three ticks and one cross", got)
	}
}

// A flow that died half way through still has to line up with the others in the
// summary, so missing operations read as failures.
func TestFormatCRUDTreatsMissingOperationsAsFailures(t *testing.T) {
	got := printing.FormatCRUD(map[string]bool{"create": true})
	if strings.Count(got, "✘") != 3 {
		t.Errorf("format = %q, want three crosses", got)
	}
}

func TestFormatCRUDKeepsExtraOperations(t *testing.T) {
	got := printing.FormatCRUD(map[string]bool{
		"create": true, "read": true, "update": true, "delete": true, "archive": true,
	})
	if !strings.Contains(got, "archive:") {
		t.Errorf("format = %q, want the extra operation kept", got)
	}
	if strings.Index(got, "D:") > strings.Index(got, "archive:") {
		t.Errorf("format = %q, want extras after the standard four", got)
	}
}

func TestGroupSeparatorIsCentredAndFixedWidth(t *testing.T) {
	var out bytes.Buffer
	printing.GroupSeparator(&out, "Run Flows")

	lines := strings.Split(strings.TrimLeft(out.String(), "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected three lines, got %q", out.String())
	}
	if len(lines[0]) != 80 || len(lines[2]) != 80 {
		t.Errorf("rules are %d/%d chars, want 80", len(lines[0]), len(lines[2]))
	}
	if len([]rune(lines[1])) != 80 {
		t.Errorf("title line is %d runes, want 80", len([]rune(lines[1])))
	}
	if !strings.Contains(lines[1], "Run Flows") {
		t.Errorf("title missing: %q", lines[1])
	}
}

// Column alignment is counted in runes, so a non-ASCII flow name doesn't shift
// the summary.
func TestPadRightCountsRunes(t *testing.T) {
	if got := printing.PadRight("café", 6); len([]rune(got)) != 6 {
		t.Errorf("PadRight = %q (%d runes), want 6 runes", got, len([]rune(got)))
	}
	if got := printing.PadRight("too long", 3); got != "too long" {
		t.Errorf("PadRight truncated: %q", got)
	}
}
