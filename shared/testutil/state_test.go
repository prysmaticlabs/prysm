package testutil

import (
	"testing"
)

func TestNewBeaconState(t *testing.T) {
	st := NewBeaconState()
	if _, err := st.InnerStateUnsafe().MarshalSSZ(); err != nil {
		t.Fatal(err)
	}
}
