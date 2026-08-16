// Package nocrud is the vocabulary a flow is written in — the types every
// other package agrees on.
//
// It sits at the bottom of the import graph on purpose. The runner imports it
// to execute flows, the helpers in utils/ import it to take a *Ctx, and your
// flow files import it to register themselves. Nothing here imports the runner
// back, which is what keeps a flow file free to use any helper it likes.
//
// The three things a flow author touches:
//
//   - Register / RegisterCRUD, called from a flow file's init()
//   - Ctx, the one argument a flow receives
//   - Must / ExpectFail / ExpectStatus, for saying what should happen
package nocrud

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Kind is what sort of flow this is, and therefore which selection flag runs
// it: -crud or -req.
//
// It is a string rather than an enum so it reads for what it is in listings and
// error messages, and so a Flow literal that forgets to set it is caught at
// registration rather than silently becoming whichever kind happened to be
// zero.
type Kind string

const (
	// CRUD is the standard create/read/update/delete check over one endpoint.
	CRUD Kind = "crud"
	// Request is everything else: multi-step, multi-user, business-logic and
	// seeding flows.
	Request Kind = "request"
)

// Valid reports whether k is a kind the runner knows.
func (k Kind) Valid() bool { return k == CRUD || k == Request }

// Func is the body of a flow.
//
// The returned value is only ever displayed — the runner formats it into the
// summary line (a crud.Result becomes "C:✔ R:✔ U:✔ D:✔", a string prints as
// itself). Failure is the error, or a panic from Must.
type Func func(c *Ctx) (any, error)

// Flow is one registered flow.
type Flow struct {
	// Name identifies the flow on the command line (-f <name>). For a CRUD
	// flow this is conventionally the endpoint, which is also what
	// cmd/modelcoverage matches models against.
	Name string
	// Kind selects which of -crud / -req runs it.
	Kind Kind
	// Doc is a one-line description, shown by -l. Optional.
	Doc string
	// Fn is the flow itself.
	Fn Func
}

// ============================================================================
//  Registry
// ============================================================================

// Flows register themselves from init(), which happens before main and
// therefore before anything can read the registry. The mutex is not for that —
// it is because tests register flows directly, and a registry that is only
// safe during init is a trap waiting for the first person who writes one.
var (
	registryMu sync.RWMutex
	registry   []Flow
	registered = map[string]bool{}
)

// Register adds a flow. Call it from an init() in your flow file:
//
//	func init() {
//		nocrud.Register(nocrud.Flow{
//			Name: "pitch_lock_after_interactions",
//			Kind: nocrud.Request,
//			Doc:  "A pitch cannot be edited once someone has commented on it",
//			Fn:   pitchLockAfterInteractionsFlow,
//		})
//	}
//
// It panics on anything that would produce a flow you can't run — no name, no
// body, an unknown kind, or a name already taken. All four are mistakes in
// source that is about to run, so failing at startup beats a flow that quietly
// never executes.
func Register(f Flow) {
	switch {
	case f.Name == "":
		panic("nocrud: flow has no Name")
	case f.Fn == nil:
		panic(fmt.Sprintf("nocrud: flow %q has no Fn", f.Name))
	case !f.Kind.Valid():
		panic(fmt.Sprintf("nocrud: flow %q has Kind %q, want %q or %q", f.Name, f.Kind, CRUD, Request))
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if registered[f.Name] {
		panic(fmt.Sprintf("nocrud: two flows are registered as %q", f.Name))
	}
	registered[f.Name] = true
	registry = append(registry, f)
}

// RegisterCRUD is the shorthand for the common case, where the flow is a CRUD
// check named after its endpoint:
//
//	func init() { nocrud.RegisterCRUD("actor", crudActorFlow) }
func RegisterCRUD(name string, fn Func) {
	Register(Flow{Name: name, Kind: CRUD, Fn: fn})
}

// All returns every registered flow, ordered by name.
//
// Sorting rather than preserving registration order matters: registration
// happens in init(), whose order across files is the compiler's business, so
// unsorted output would reshuffle for reasons that have nothing to do with the
// flows.
func All() []Flow {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := append([]Flow(nil), registry...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ByKind returns the registered flows of one kind, ordered by name.
func ByKind(kind Kind) []Flow {
	var out []Flow
	for _, f := range All() {
		if f.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}

// Select resolves flow names to flows, in the order asked for.
//
// Every unknown name is reported at once, with the registered names alongside
// — a typo should not cost you a second run to discover the next one.
func Select(names []string) ([]Flow, error) {
	byName := map[string]Flow{}
	for _, f := range All() {
		byName[f.Name] = f
	}

	var out []Flow
	var missing []string
	for _, name := range names {
		f, ok := byName[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		out = append(out, f)
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("no flow named %s\n\nRegistered flows:\n  %s",
			strings.Join(quoteAll(missing), ", "), strings.Join(Names(All()), "\n  "))
	}
	return out, nil
}

// Names returns the names of the given flows.
func Names(flows []Flow) []string {
	out := make([]string, len(flows))
	for i, f := range flows {
		out[i] = f.Name
	}
	return out
}

func quoteAll(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = fmt.Sprintf("%q", v)
	}
	return out
}
