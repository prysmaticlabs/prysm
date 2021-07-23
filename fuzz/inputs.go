package fuzz

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
)

// InputBlockWithPrestate for fuzz testing beacon blocks.
type InputBlockWithPrestate struct {
	State *statepb.BeaconState
	Block *ethpb.SignedBeaconBlock
}
