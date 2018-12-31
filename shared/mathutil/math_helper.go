package mathutil

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
	return (n & (n - 1)) == 0
}
