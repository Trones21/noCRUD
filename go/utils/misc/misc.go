// Package misc holds the small odds and ends — the Go counterpart of
// python/utils/misc.py.
package misc

import (
	"math/rand"
)

const alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString produces a random string of n characters.
//
// If you want a random string of varying length, make sure it is never too
// short when it lands on a field with a unique constraint.
func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphanum[rand.Intn(len(alphanum))]
	}
	return string(b)
}
