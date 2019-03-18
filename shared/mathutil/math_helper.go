package mathutil

import "math"

// IntegerSquareRoot defines a function that returns the
// largest possible integer root of a number.
func IntegerSquareRoot(n uint64) uint64 {
	x := n
	y := (x + 1) / 2

	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
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

// ClosestPowerOf2 returns an integer that is the closest
// power of 2 that is less than or equal to the argument.
func ClosestPowerOf2(n uint64) uint64 {
	if n == 0 {
		return uint64(1)
	}
	exponent := math.Floor(math.Log2(float64(n)))
	return PowerOf2(uint64(exponent))
}
