package fuzz

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// InputBlockWithPrestate for fuzz testing beacon blocks.
type InputBlockWithPrestate struct {
	State *pb.BeaconState
	Block *ethpb.SignedBeaconBlock
}
