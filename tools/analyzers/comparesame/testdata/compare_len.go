package testdata

func Equal() {
	x := []string{"a"}
	if len(x) == len(x) { // want "Boolean expression has identical expressions on both sides. The result is always true."
	}
}

func NotEqual() {
	x := []string{"a"}
	if len(x) != len(x) { // want "Boolean expression has identical expressions on both sides. The result is always true."
	}
}

func GreaterThanOrEqual() {
	x := []string{"a"}
	if len(x) >= len(x) { // want "Boolean expression has identical expressions on both sides. The result is always true."
	}
}

func LessThanOrEqual() {
	x := []string{"a"}
	if len(x) <= len(x) { // want "Boolean expression has identical expressions on both sides. The result is always true."
	}
}

func GreaterThan() {
	x := []string{"a"}
	if len(x) > len(x) { // want "Boolean expression has identical expressions on both sides. The result is always false."
	}
}

func LessThan() {
	x := []string{"a"}
	if len(x) < len(x) { // want "Boolean expression has identical expressions on both sides. The result is always false."
	}
}
