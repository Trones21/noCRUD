// Command createcrudflow generates a CRUD flow file — the Go counterpart of
// python/create_crud_flow.py.
//
// Example: create a CRUD flow for a `book` entity/url, setting the title field
// to "newtitle" on the update step.
//
//	go run ./cmd/createcrudflow -uf title -ufv newtitle book
//
// Please ensure that the url is correct, the fixture exists (and the name
// matches), and the field exists on the model.
//
// Unlike the Python generator, the flags come before the object name: Go's flag
// package stops parsing at the first non-flag argument.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/Trones21/noCRUD/go/config"
)

const flowTemplate = `package flows

import (
	"github.com/Trones21/noCRUD/go/nocrud"
	"github.com/Trones21/noCRUD/go/utils/crud"
)

func init() { nocrud.RegisterCRUD("{{.Object}}", crud{{.Pascal}}Flow) }

// crud{{.Pascal}}Flow runs create/read/update/delete against the {{.Object}} endpoint.
func crud{{.Pascal}}Flow(c *nocrud.Ctx) (any, error) {
	api, err := c.Setup()
	if err != nil {
		return nil, err
	}

	return crud.Exec(c, api, "{{.Object}}",
		// Swap SimpleCreate for your own crud.CreateFunc when the object needs
		// other objects to exist first.
		crud.SimpleCreate("{{.Object}}", "{{.Fixture}}", 0, "id"),
		crud.UpdateDetails{Field: "{{.UpdateField}}", NewValue: "{{.FieldValue}}"},
	)
}
`

type flowData struct {
	Object      string
	Pascal      string
	Fixture     string
	UpdateField string
	FieldValue  string
}

func main() {
	updateField := flag.String("uf", "", "The field to update in the object")
	flag.StringVar(updateField, "updateField", "", "The field to update in the object")
	fieldValue := flag.String("ufv", "", "The new value to set for the field")
	flag.StringVar(fieldValue, "fieldval", "", "The new value to set for the field")
	fixture := flag.String("fixture", "", "Fixture file to create from (default: <obj_name>s.json)")
	dir := flag.String("path", "", "Directory to write the file to (default: <runner>/flows)")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Generate a CRUD flow file with an update step.\n\nUsage:\n  %s -uf <field> -ufv <value> <obj_name>\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	object := flag.Arg(0)
	if object == "" || *updateField == "" || *fieldValue == "" {
		flag.Usage()
		os.Exit(2)
	}

	data := flowData{
		Object:      object,
		Pascal:      snakeToPascal(object),
		Fixture:     defaultString(*fixture, object+"s.json"),
		UpdateField: *updateField,
		FieldValue:  *fieldValue,
	}

	baseDir := defaultString(*dir, filepath.Join(config.RunnerDir(), "flows"))
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	path := filepath.Join(baseDir, object+".go")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "❌ %s already exists — refusing to overwrite it\n", path)
		os.Exit(1)
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	if err := template.Must(template.New("flow").Parse(flowTemplate)).Execute(f, data); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("✅ File created: %s\n", path)
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

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
