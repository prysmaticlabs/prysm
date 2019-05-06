package rpc

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// AttesterServer defines a server implementation of the gRPC Attester service,
// providing RPC methods for validators acting as attesters to broadcast votes on beacon blocks.
type AttesterServer struct {
	p2p              p2p.Broadcaster
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

	if err := as.operationService.HandleAttestations(ctx, att); err != nil {
		return nil, err
	}
	as.p2p.Broadcast(ctx, &pbp2p.AttestationAnnounce{
		Hash: h[:],
	})
	return &pb.AttestResponse{AttestationHash: h[:]}, nil
}

// AttestationDataAtSlot fetches the necessary information from the current canonical head
// and beacon state for an assigned attester to perform necessary responsibilities. This includes
// fetching the epoch boundary roots, the latest justified block root, among others.
func (as *AttesterServer) AttestationDataAtSlot(ctx context.Context, req *pb.AttestationDataRequest) (*pb.AttestationDataResponse, error) {
	// Set the attestation data's beacon block root = hash_tree_root(head) where head
	// is the validator's view of the head block of the beacon chain during the slot.
	head, err := as.beaconDB.ChainHead()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chain head: %v", err)
	}
	headRoot, err := hashutil.HashBeaconBlock(head)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash beacon block: %v", err)
	}

	// Let head state be the state of head block processed through empty slots up to assigned slot.
	headState, err := as.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch head state: %v", err)
	}

	for headState.Slot < req.Slot {
		headState, err = state.ExecuteStateTransition(
			ctx, headState, nil /* block */, headRoot, state.DefaultConfig(),
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute head transition: %v", err)
		}
	}

	// Fetch the epoch boundary root = hash_tree_root(epoch_boundary)
	// where epoch_boundary is the block at the most recent epoch boundary in the
	// chain defined by head -- i.e. the BeaconBlock where block.slot == get_epoch_start_slot(head.slot).
	// If the epoch boundary slot is the same as state current slot,
	// we set epoch boundary root to an empty root.
	epochBoundaryRoot := make([]byte, 32)
	epochStartSlot := helpers.StartSlot(helpers.SlotToEpoch(headState.Slot))
	if epochStartSlot == headState.Slot {
		epochBoundaryRoot = headRoot[:]
	} else {
		epochBoundaryRoot, err = blocks.BlockRoot(headState, epochStartSlot)
		if err != nil {
			return nil, fmt.Errorf("could not get epoch boundary block for slot %d: %v",
				epochStartSlot, err)
		}
	}
	// epoch_start_slot = get_epoch_start_slot(slot_to_epoch(head.slot))
	// Fetch the justified block root = hash_tree_root(justified_block) where
	// justified_block is the block at state.justified_epoch in the chain defined by head.
	// On the server side, this is fetched by calling get_block_root(state, justified_epoch).
	// If the last justified boundary slot is the same as state current slot (ex: slot 0),
	// we set justified block root to an empty root.
	justifiedBlockRoot := headState.JustifiedRoot

	// If an attester has to attest for genesis block.
	if headState.Slot == params.BeaconConfig().GenesisSlot {
		epochBoundaryRoot = params.BeaconConfig().ZeroHash[:]
		justifiedBlockRoot = params.BeaconConfig().ZeroHash[:]
	}

	return &pb.AttestationDataResponse{
		HeadSlot:                 headState.Slot,
		BeaconBlockRootHash32:    headRoot[:],
		EpochBoundaryRootHash32:  epochBoundaryRoot,
		JustifiedEpoch:           headState.JustifiedEpoch,
		JustifiedBlockRootHash32: justifiedBlockRoot,
		LatestCrosslink:          headState.LatestCrosslinks[req.Shard],
	}, nil
}
