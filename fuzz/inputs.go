package fuzz

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// InputBlockWithPrestate for fuzz testing beacon blocks.
type InputBlockWithPrestate struct {
	State *ethpb.BeaconState
	Block *ethpb.SignedBeaconBlock
}
