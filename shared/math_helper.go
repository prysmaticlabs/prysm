package shared

func IntegerSquareRoot(n uint64) uint64 {
	x := n
	y := (x + 1) / 2

	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}
