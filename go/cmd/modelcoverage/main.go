// Command modelcoverage reports which of the app's models have a flow — the Go
// counterpart of python/model_coverage_check.py.
//
//	go run ./cmd/modelcoverage -models ../example_app/api/models
//
// Where the Python version compares model names against the *filenames* in
// flows/crud/, this one compares them against the flows that actually
// registered. A file that exists but never registered its flow is exactly the
// mistake worth catching, and only the registry can see it.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/Trones21/noCRUD/go/config"
	"github.com/Trones21/noCRUD/go/nocrud"

	_ "github.com/Trones21/noCRUD/go/flows"
)

// manualMapping records models handled somewhere other than a same-named flow.
// Key: model name. Value: where it's handled, or "" if that's still an open
// question. Edit this in your own copy.
var manualMapping = map[string]string{}

// classPattern matches a Django model declaration: a class whose bases include
// something ending in Model (models.Model, Model, TimeStampedModel...).
var classPattern = regexp.MustCompile(`(?m)^\s*class\s+(\w+)\s*\(([^)]*)\)\s*:`)

func main() {
	modelsDir := flag.String("models", "", "Directory holding the app's model definitions (default: <app>/api/models)")
	flag.Parse()

	dir := *modelsDir
	if dir == "" {
		dir = filepath.Join(config.AppDir(), "api", "models")
	}

	models, err := modelNames(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	covered := flowTargets()

	fmt.Println("=== Model Flow Coverage ===")
	fmt.Println()

	missing := 0
	for _, model := range sorted(models) {
		if where, ok := manualMapping[model]; ok {
			fmt.Printf("⏭️  %s — manually handled: %s\n", model, defaultString(where, "(pending)"))
			continue
		}
		if covered[model] {
			fmt.Printf("✅ %s\n", model)
			continue
		}
		fmt.Printf("❌ %s — no corresponding flow\n", model)
		missing++
	}

	// Flows whose name matches no model — usually a typo, sometimes a
	// business-logic flow that was registered as CRUD.
	var extra []string
	for target := range covered {
		if !models[target] && manualMapping[target] == "" {
			extra = append(extra, target)
		}
	}
	if len(extra) > 0 {
		fmt.Println("\n⚠️ Flows with no matching model:")
		for _, flow := range sorted(setOf(extra)) {
			fmt.Printf("🟡 %s\n", flow)
		}
	}

	if len(manualMapping) > 0 {
		fmt.Println("\n=== Manual Mapping Summary ===")
		for model, where := range manualMapping {
			status := "✅"
			if where == "" {
				status = "❌"
				where = "(empty — needs attention)"
			}
			fmt.Printf("%s %s: %s\n", status, model, where)
		}
	}

	if missing > 0 {
		os.Exit(1)
	}
}

// modelNames scans a directory of Python model definitions.
//
// The Python version parses the files with ast; matching the class declaration
// is the part of that which actually decides the answer, and it doesn't need a
// Python interpreter in the loop.
func modelNames(dir string) (map[string]bool, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.py"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no model files found in %s (pass -models)", dir)
	}

	models := map[string]bool{}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		for _, match := range classPattern.FindAllStringSubmatch(string(src), -1) {
			name, bases := match[1], match[2]
			for _, base := range strings.Split(bases, ",") {
				base = strings.TrimSpace(base)
				if base == "Model" || strings.HasSuffix(base, ".Model") {
					models[name] = true
					break
				}
			}
		}
	}
	return models, nil
}

// flowTargets returns the registered CRUD flow names, as model names would be
// written: actor → Actor, tag_category → TagCategory.
func flowTargets() map[string]bool {
	targets := map[string]bool{}
	for _, flow := range nocrud.ByKind(nocrud.CRUD) {
		targets[snakeToPascal(flow.Name)] = true
	}
	return targets
}

func snakeToPascal(s string) string {
	var b strings.Builder
	for _, word := range strings.Split(s, "_") {
		if word == "" {
			continue
		}
		runes := []rune(word)
		b.WriteRune(unicode.ToUpper(runes[0]))
		b.WriteString(string(runes[1:]))
	}
	return b.String()
}

func sorted(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func setOf(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
