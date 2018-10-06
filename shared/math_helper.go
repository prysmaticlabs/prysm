package shared

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
