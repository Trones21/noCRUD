package jsonx_test

import (
	"testing"

	"github.com/Trones21/noCRUD/go/utils/jsonx"
)

const paginated = `{"count": 2, "results": [{"id": 7, "username": "var_undecided", "active": true}, {"id": 8}]}`

func TestGetWalksObjectsAndArrays(t *testing.T) {
	v := jsonx.Unmarshal([]byte(paginated))

	if got := v.Get("results.0.username").Text(); got != "var_undecided" {
		t.Errorf("username = %q", got)
	}
	if got := v.Get("results.0.id").Int(); got != 7 {
		t.Errorf("id = %d, want 7", got)
	}
	if got := v.Get("count").Int(); got != 2 {
		t.Errorf("count = %d, want 2", got)
	}
	if !v.Get("results.0.active").Bool() {
		t.Error("active = false, want true")
	}
	if got := v.Get("results").Len(); got != 2 {
		t.Errorf("len(results) = %d, want 2", got)
	}
	if got := len(v.Get("results").Items()); got != 2 {
		t.Errorf("Items() returned %d", got)
	}
}

// A miss has to be inert: a flow reading an optional field shouldn't have to
// guard every lookup, and it certainly shouldn't take the run down.
func TestMissesReturnErrorsRatherThanPanicking(t *testing.T) {
	v := jsonx.Unmarshal([]byte(paginated))

	for name, got := range map[string]jsonx.Value{
		"missing key":    v.Get("nope"),
		"index too high": v.Get("results.9"),
		"not an index":   v.Get("results.abc"),
		"not an object":  v.Get("count.id"),
		"nested miss":    v.Get("results.0.nope.deeper"),
	} {
		if got.Err() == nil {
			t.Errorf("%s: expected an error", name)
		}
		if got.Exists() {
			t.Errorf("%s: Exists() = true", name)
		}
		// Accessors on a failed lookup are zero values, not panics.
		if got.Text() != "" || got.Int() != 0 || got.Bool() || got.Map() != nil || got.Slice() != nil {
			t.Errorf("%s: accessors did not return zero values", name)
		}
	}
}

func TestIDRendersIdentifiersUsableInAURL(t *testing.T) {
	cases := map[string]struct {
		raw  any
		want string
	}{
		"integer pk":     {float64(42), "42"},
		"large integer":  {float64(1234567890123), "1234567890123"},
		"uuid":           {"3f2504e0-4f89-11d3-9a0c-0305e82c3301", "3f2504e0-4f89-11d3-9a0c-0305e82c3301"},
		"missing":        {nil, ""},
		"decimal number": {1.5, "1.5"},
	}
	for name, tc := range cases {
		if got := jsonx.New(tc.raw).ID(); got != tc.want {
			t.Errorf("%s: ID() = %q, want %q", name, got, tc.want)
		}
	}
}

func TestSetMutatesTheUnderlyingObject(t *testing.T) {
	v := jsonx.Unmarshal([]byte(`{"name": "old"}`))
	v.Set("name", "new")

	if got := v.Get("name").Text(); got != "new" {
		t.Errorf("name = %q, want new", got)
	}
	// Setting on a non-object is a no-op rather than a panic.
	jsonx.New(float64(1)).Set("name", "x")
}

func TestStringRendersCompactJSON(t *testing.T) {
	v := jsonx.Unmarshal([]byte(`{"id":1}`))
	if got := v.String(); got != `{"id":1}` {
		t.Errorf("String() = %q", got)
	}
	if got := jsonx.Unmarshal([]byte("not json")).Err(); got == nil {
		t.Error("expected invalid JSON to be reported")
	}
}
