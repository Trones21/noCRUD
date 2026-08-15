// Package jsonx is a thin ergonomic wrapper over decoded JSON (any).
//
// The Python runner hands flows a plain dict, and a flow reads it with
// res["results"][0]["id"]. The literal Go translation of that is a pile of type
// assertions, which is exactly the verbosity that makes a Go runner unpleasant
// to write flows in. Value gives back the one-liner:
//
//	res.Get("results.0.id").ID()
//
// Lookups never panic. A miss (wrong key, wrong type, index out of range)
// yields a zero Value carrying an error, which you can check with Err() when it
// matters and ignore when it doesn't.
package jsonx

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Value wraps a decoded JSON value.
type Value struct {
	raw any
	err error
}

// New wraps an already-decoded value.
func New(raw any) Value { return Value{raw: raw} }

// Invalid returns a Value carrying err. Accessors on it return zero values.
func Invalid(err error) Value { return Value{err: err} }

// Unmarshal decodes JSON bytes into a Value.
func Unmarshal(b []byte) Value {
	var raw any
	if err := json.Unmarshal(b, &raw); err != nil {
		return Invalid(err)
	}
	return Value{raw: raw}
}

// Raw returns the underlying decoded value (map[string]any, []any, string,
// float64, bool or nil).
func (v Value) Raw() any { return v.raw }

// Err reports why a lookup failed, or nil.
func (v Value) Err() error { return v.err }

// Exists reports whether the value is present (no lookup error and not null).
func (v Value) Exists() bool { return v.err == nil && v.raw != nil }

// Get walks a dotted path. Path segments are object keys, except for
// all-digit segments, which index into arrays: "results.0.id".
func (v Value) Get(path string) Value {
	if v.err != nil {
		return v
	}
	cur := v.raw
	for _, seg := range strings.Split(path, ".") {
		if seg == "" {
			continue
		}
		switch node := cur.(type) {
		case map[string]any:
			next, ok := node[seg]
			if !ok {
				return Invalid(fmt.Errorf("jsonx: key %q not found in %q", seg, path))
			}
			cur = next
		case []any:
			i, err := strconv.Atoi(seg)
			if err != nil {
				return Invalid(fmt.Errorf("jsonx: %q is not an array index (path %q)", seg, path))
			}
			if i < 0 || i >= len(node) {
				return Invalid(fmt.Errorf("jsonx: index %d out of range (len %d, path %q)", i, len(node), path))
			}
			cur = node[i]
		default:
			return Invalid(fmt.Errorf("jsonx: cannot descend into %T at %q (path %q)", cur, seg, path))
		}
	}
	return Value{raw: cur}
}

// Text returns the value as a string. Non-strings yield "".
func (v Value) Text() string {
	s, _ := v.raw.(string)
	return s
}

// Float returns the value as a float64. Non-numbers yield 0.
func (v Value) Float() float64 {
	switch n := v.raw.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}

// Int returns the value as an int. Non-numbers yield 0.
func (v Value) Int() int { return int(v.Float()) }

// Bool returns the value as a bool. Non-bools yield false.
func (v Value) Bool() bool {
	b, _ := v.raw.(bool)
	return b
}

// ID renders an identifier as a string, whether the API returned it as a JSON
// number (integer pk) or a string (uuid, slug). Whole numbers never come back
// in exponent notation, so the result is always safe to splice into a URL.
func (v Value) ID() string {
	switch n := v.raw.(type) {
	case string:
		return n
	case float64:
		if n == float64(int64(n)) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'f', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprint(n)
	}
}

// Map returns the value as an object. Non-objects yield nil.
func (v Value) Map() map[string]any {
	m, _ := v.raw.(map[string]any)
	return m
}

// Slice returns the value as an array. Non-arrays yield nil.
func (v Value) Slice() []any {
	s, _ := v.raw.([]any)
	return s
}

// Len returns the number of elements in an array or keys in an object.
func (v Value) Len() int {
	switch n := v.raw.(type) {
	case []any:
		return len(n)
	case map[string]any:
		return len(n)
	}
	return 0
}

// Items returns an array's elements as Values, so they can be ranged over
// without re-wrapping.
func (v Value) Items() []Value {
	raw := v.Slice()
	out := make([]Value, len(raw))
	for i, item := range raw {
		out[i] = Value{raw: item}
	}
	return out
}

// Set writes a key on an object value, so a flow can round-trip an object it
// just read (GET, change one field, PUT). It is a no-op on non-objects.
func (v Value) Set(key string, val any) Value {
	if m := v.Map(); m != nil {
		m[key] = val
	}
	return v
}

// String renders the value as compact JSON, so a Value prints readably with
// %v/%s the way the Python runner's dicts do.
func (v Value) String() string {
	if v.err != nil {
		return fmt.Sprintf("<jsonx error: %v>", v.err)
	}
	b, err := json.Marshal(v.raw)
	if err != nil {
		return fmt.Sprint(v.raw)
	}
	return string(b)
}

// Pretty renders the value as indented JSON (used when printing error bodies).
func (v Value) Pretty() string {
	b, err := json.MarshalIndent(v.raw, "", "  ")
	if err != nil {
		return fmt.Sprint(v.raw)
	}
	return string(b)
}
