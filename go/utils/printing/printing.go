// Package printing holds the output formatting — the Go counterpart of
// python/utils/printing.py.
package printing

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
)

const (
	tick  = "\033[92m✔\033[0m"
	cross = "\033[91m✘\033[0m"
)

// crudOrder is the order the four operations are printed in. Any extra keys a
// flow adds to its result are printed after these, alphabetically.
var crudOrder = []string{"create", "read", "update", "delete"}

// FormatCRUD renders a CRUD result map as "C:✔ R:✔ U:✔ D:✔".
//
// A key that is missing from the map renders as a failure, so a flow that died
// half way through still lines up with the others in the summary table.
func FormatCRUD(res map[string]bool) string {
	mark := func(ok bool) string {
		if ok {
			return tick
		}
		return cross
	}

	var parts []string
	for _, k := range crudOrder {
		parts = append(parts, fmt.Sprintf("%s:%s", strings.ToUpper(k[:1]), mark(res[k])))
	}

	// Anything the flow added beyond the four standard operations.
	var extra []string
	for k := range res {
		if !slices.Contains(crudOrder, k) {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		parts = append(parts, fmt.Sprintf("%s:%s", k, mark(res[k])))
	}

	return strings.Join(parts, " ")
}

// GroupSeparator prints a centered banner, matching the Python runner's
// print_group_separator.
func GroupSeparator(w io.Writer, text string) {
	fmt.Fprintln(w, "\n"+strings.Repeat("=", 80))
	fmt.Fprintln(w, center(text, 80))
	fmt.Fprintln(w, strings.Repeat("=", 80))
}

// Warn prints a warning banner, matching the Python runner's print_warn.
func Warn(w io.Writer, text string) {
	fmt.Fprintln(w, "\n"+strings.Repeat("=", 33)+" WARNING ⚠️  "+strings.Repeat("=", 34))
	fmt.Fprintln(w, center(text, 79))
	fmt.Fprintln(w, strings.Repeat("=", 79))
}

// Rule prints a plain horizontal rule.
func Rule(w io.Writer, n int) {
	fmt.Fprintln(w, strings.Repeat("=", n))
}

// center pads text to width, centered, the way Python's str.center does. Width
// is counted in runes so the banners line up when the text has non-ASCII in it.
func center(text string, width int) string {
	n := len([]rune(text))
	if n >= width {
		return text
	}
	left := (width - n) / 2
	right := width - n - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

// PadRight pads s to width with spaces, counting runes rather than bytes so
// summary columns stay aligned with non-ASCII flow names.
func PadRight(s string, width int) string {
	if n := len([]rune(s)); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}
