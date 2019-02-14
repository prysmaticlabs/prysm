package rpc

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// AttesterServer defines a server implementation of the gRPC Attester service,
// providing RPC methods for validators acting as attesters to broadcast votes on beacon blocks.
type AttesterServer struct {
	beaconDB         *db.BeaconDB
	operationService operationService
}

// AttestHead is a function called by an attester in a sharding validator to vote
// on a block via an attestation object as defined in the Ethereum Serenity specification.
func (as *AttesterServer) AttestHead(ctx context.Context, att *pbp2p.Attestation) (*pb.AttestResponse, error) {
	h, err := hashutil.HashProto(att)
	if err != nil {
		return nil, fmt.Errorf("could not hash attestation: %v", err)
	}
	// Relays the attestation to chain service.
	as.operationService.IncomingAttFeed().Send(att)
	return &pb.AttestResponse{AttestationHash: h[:]}, nil
}

// AttestationInfoAtSlot fetches the necessary information from the current canonical head
// and beacon state for an assigned attester to perform necessary responsibilities. This includes
// fetching the epoch boundary roots, the latest justified block root, among others.
func (as *AttesterServer) AttestationInfoAtSlot(ctx context.Context, req *pb.AttestationInfoRequest) (*pb.AttestationInfoResponse, error) {
	// Set the attestation data's beacon block root = hash_tree_root(head) where head
	// is the validator's view of the head block of the beacon chain during the slot.
	head, err := as.beaconDB.BlockBySlot(req.Slot)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chain head: %v", err)
	}
	if head == nil {
		return nil, fmt.Errorf("no block found at slot %d", req.Slot)
	}
	blockRoot, err := ssz.TreeHash(head)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash beacon block: %v", err)
	}
	beaconState, err := as.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	// Fetch the epoch boundary root = hash_tree_root(epoch_boundary)
	// where epoch_boundary is the block at the most recent epoch boundary in the
	// chain defined by head -- i.e. the BeaconBlock where block.slot == get_epoch_start_slot(head.slot).
	// On the server side, this is fetched by calling get_block_root(state, get_epoch_start_slot(head.slot)).
	epochBoundaryRoot, err := blocks.BlockRoot(beaconState, helpers.StartSlot(head.Slot))
	if err != nil {
		return nil, fmt.Errorf("could not get epoch boundary block: %v", err)
	}
	// Fetch the justified block root = hash_tree_root(justified_block) where
	// justified_block is the block at state.justified_epoch in the chain defined by head.
	// On the server side, this is fetched by calling get_block_root(state, justified_epoch).
	justifiedBlockRoot, err := blocks.BlockRoot(beaconState, beaconState.JustifiedEpoch)
	if err != nil {
		return nil, fmt.Errorf("could not get justified block: %v", err)
	}
	return &pb.AttestationInfoResponse{
		BeaconBlockRootHash32:    blockRoot[:],
		EpochBoundaryRootHash32:  epochBoundaryRoot[:],
		JustifiedEpoch:           beaconState.JustifiedEpoch,
		JustifiedBlockRootHash32: justifiedBlockRoot[:],
		LatestCrosslink:          beaconState.LatestCrosslinks[req.Shard],
	}, nil
}
