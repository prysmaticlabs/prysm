package fuzz

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v2"
)

// InputBlockWithPrestate for fuzz testing beacon blocks.
type InputBlockWithPrestate struct {
	State *pb.BeaconState
	Block *ethpb.SignedBeaconBlock
}
