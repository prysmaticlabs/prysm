package store

import (
	"sync"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// store is defined in the fork choice consensus spec for tracking current time and various versions of checkpoints.
//
// Spec code:
// class Store(object):
//    time: uint64
//    genesis_time: uint64
//    justified_checkpoint: Checkpoint
//    finalized_checkpoint: Checkpoint
//    best_justified_checkpoint: Checkpoint
//    proposerBoostRoot: Root
type store struct {
	time                 uint64
	genesisTime          uint64
	justifiedCheckpt     *ethpb.Checkpoint
	finalizedCheckpt     *ethpb.Checkpoint
	bestJustifiedCheckpt *ethpb.Checkpoint
	sync.RWMutex
	// These are not part of the consensus spec, but we do use them to return gRPC API requests.
	// I highly doubt anyone uses them, we can consider removing them in v3.
	prevFinalizedCheckpt *ethpb.Checkpoint
	prevJustifiedCheckpt *ethpb.Checkpoint
}
