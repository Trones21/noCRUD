package nocrud

import (
	"fmt"

	"github.com/Trones21/noCRUD/go/utils/apiclient"
)

// FlowPanic is a failure a flow raised on purpose, via Must.
//
// The runner unwraps it, so a Must that trips reads as a plain failure rather
// than a crash — the same way a Python flow signals failure by letting an
// exception escape. An accidental panic (nil map, index out of range) is not
// one of these and does get reported as a panic, with its stack.
type FlowPanic struct{ Err error }

func (p *FlowPanic) Error() string { return p.Err.Error() }
func (p *FlowPanic) Unwrap() error { return p.Err }

// Must returns v, or fails the flow if err is non-nil.
//
// It exists because Go's error handling is at odds with what a flow is. Three
// lines per call is right for a program that has to keep running; a flow that
// can't create the object it was about to use has nothing left to do, and
// wrapping every step in "if err != nil { return nil, err }" buries the flow
// under its own plumbing:
//
//	created := nocrud.Must(api.CreateObject("universe", obj))
//
// Use it for the steps that are setup rather than the point — the object graph
// a rule needs before the rule can be tested. Where the failure *is* the
// interesting part, return the error (or use ExpectFail), because that reads
// as deliberate to whoever finds it later.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(&FlowPanic{Err: err})
	}
	return v
}

// ExpectFail asserts that something is refused, without saying how.
//
// The description is what shows up when the expectation doesn't hold, so write
// it as the thing being attempted — "editing a locked pitch":
//
//	err := nocrud.ExpectFail(c, "editing a locked pitch", func() error {
//		_, err := author.UpdateObjectByID("pitch", id, edit)
//		return err
//	})
//
// Prefer ExpectStatus when you know which status is correct. This is for the
// cases where the backend's failure mode is uglier than it should be — a
// ValidationError raised from save() surfaces as a 500 rather than a 400 — and
// pinning the status would assert the bug instead of the rule.
func ExpectFail(c *Ctx, description string, fn func() error) error {
	err := fn()
	if err == nil {
		return fmt.Errorf("expected %s to fail, but it succeeded", description)
	}
	c.Printf("✅ %s failed as expected: %v\n", description, err)
	return nil
}

// ExpectStatus asserts that something is refused with a particular HTTP status.
//
//	err := nocrud.ExpectStatus(c, "another user editing it", 403, func() error {
//		_, err := other.UpdateObjectByID("universe", id, edit)
//		return err
//	})
//
// Both ways of not meeting the expectation are reported, and they mean
// different things: succeeding outright is a missing rule, while failing with
// the wrong status is usually a rule enforced in the wrong layer (a 500 where
// the permission check should have produced a 403, a 404 hiding a 403).
func ExpectStatus(c *Ctx, description string, status int, fn func() error) error {
	err := fn()
	if err == nil {
		return fmt.Errorf("expected %s to fail with %d, but it succeeded", description, status)
	}

	got := apiclient.StatusOf(err)
	if got != status {
		if got == 0 {
			return fmt.Errorf("expected %s to fail with %d, but it failed without a status: %w", description, status, err)
		}
		return fmt.Errorf("expected %s to fail with %d, got %d: %w", description, status, got, err)
	}

	c.Printf("✅ %s was refused with %d, as expected\n", description, status)
	return nil
}
