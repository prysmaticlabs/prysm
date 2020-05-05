package testing

import (
	"testing"
)

func TestGetBeaconFuzzState(t *testing.T) {
	if _, err := GetBeaconFuzzState(1); err != nil {
		t.Fatal(err)
	}
}
