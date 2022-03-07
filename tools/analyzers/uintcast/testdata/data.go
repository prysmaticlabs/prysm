package testdata

import "math"

// Uint64CastToInt --
func Uint64CastToInt() {
	a := uint64(math.MaxUint64)
	b := int(a) // want "Unsafe cast from uint64 to int."

	_ = b
}

// Uint64CastToIntIfStatement --
func Uint64CastToIntIfStatement() {
	var b []string
	a := uint64(math.MaxUint64)

	if len(b) < int(a) { // want "Unsafe cast from uint64 to int."
		return
	}
}
