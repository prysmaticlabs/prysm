package testutil

import (
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestNewBeaconState(t *testing.T) {
	st := NewBeaconState()
	b, err := st.InnerStateUnsafe().MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	got := &pb.BeaconState{}
	if err := got.UnmarshalSSZ(b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(st.InnerStateUnsafe(), got) {
		t.Fatal("State did not match after round trip marshal")
	}
}
