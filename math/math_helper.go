// Package math includes important helpers for Ethereum such as fast integer square roots.
package math

import (
	"errors"
	stdmath "math"
	"math/bits"

	"github.com/thomaso-mirodin/intmath/u64"
)

// Common square root values.
var squareRootTable = map[uint64]uint64{
	4:       2,
	16:      4,
	64:      8,
	256:     16,
	1024:    32,
	4096:    64,
	16384:   128,
	65536:   256,
	262144:  512,
	1048576: 1024,
	4194304: 2048,
}

// IntegerSquareRoot defines a function that returns the
// largest possible integer root of a number using go's standard library.
func IntegerSquareRoot(n uint64) uint64 {
	if v, ok := squareRootTable[n]; ok {
		return v
	}

	// Golang floating point precision may be lost above 52 bits, so we use a
	// non floating point method. u64.Sqrt is about x2.5 slower than math.Sqrt.
	if n >= 1<<52 {
		return u64.Sqrt(n)
	}

	return uint64(stdmath.Sqrt(float64(n)))
}

// CeilDiv8 divides the input number by 8
// and takes the ceiling of that number.
func CeilDiv8(n int) int {
	ret := n / 8
	if n%8 > 0 {
		ret++
	}

	return ret
}

// IsPowerOf2 returns true if n is an
// exact power of two. False otherwise.
func IsPowerOf2(n uint64) bool {
	return n != 0 && (n&(n-1)) == 0
}

// PowerOf2 returns an integer that is the provided
// exponent of 2. Can only return powers of 2 till 63,
// after that it overflows
func PowerOf2(n uint64) uint64 {
	if n >= 64 {
		panic("integer overflow")
	}
	return 1 << n
}

// Max returns the larger integer of the two
// given ones.This is used over the Max function
// in the standard math library because that max function
// has to check for some special floating point cases
// making it slower by a magnitude of 10.
func Max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller integer of the two
// given ones. This is used over the Min function
// in the standard math library because that min function
// has to check for some special floating point cases
// making it slower by a magnitude of 10.
func Min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Mul64 multiples 2 64-bit unsigned integers and checks if they
// lead to an overflow. If they do not, it returns the result
// without an error.
func Mul64(a, b uint64) (uint64, error) {
	overflows, val := bits.Mul64(a, b)
	if overflows > 0 {
		return 0, errors.New("multiplication overflows")
	}
	return val, nil
}

// Add64 adds 2 64-bit unsigned integers and checks if they
// lead to an overflow. If they do not, it returns the result
// without an error.
func Add64(a, b uint64) (uint64, error) {
	res, carry := bits.Add64(a, b, 0 /* carry */)
	if carry > 0 {
		return 0, errors.New("addition overflows")
	}
	return res, nil
}

// Sub64 subtracts two 64-bit unsigned integers and checks for errors.
func Sub64(a, b uint64) (uint64, error) {
	res, borrow := bits.Sub64(a, b, 0 /* borrow */)
	if borrow > 0 {
		return 0, errors.New("subtraction underflow")
	}
	return res, nil
}
