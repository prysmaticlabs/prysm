package bls

import "testing"

func TestGetBLS12381Curve(t *testing.T) {
	if 5 != getBLS12381Curve() {
		t.Errorf("Wrong curve int: expected %v, received %v", 5, getBLS12381Curve())
	}
}
