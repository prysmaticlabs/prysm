package helpers

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func TestSigningRoot_ComputeOK(t *testing.T) {
	emptyBlock := &ethpb.BeaconBlock{}
	_, err := ComputeSigningRoot(emptyBlock, []byte{'T', 'E', 'S', 'T'})
	if err != nil {
		t.Errorf("Could not compute signing root of block: %v", err)
	}
}
