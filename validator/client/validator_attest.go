package client

import (
	"context"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/opentracing/opentracing-go"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// AttestToBlockHead
//
// WIP - not done.
func (v *validator) AttestToBlockHead(ctx context.Context, slot uint64) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.AttestToBlockHead")
	defer span.Finish()
	// First the validator should construct attestation_data, an AttestationData
	// object based upon the state at the assigned slot.
	attData := &pbp2p.AttestationData{
		Slot:                 slot,
		ShardBlockRootHash32: params.BeaconConfig().ZeroHash[:], // Stub for Phase 0.
	}
	req := &pb.CrosslinkCommitteeRequest{
		Slot: slot,
	}
	resp, err := v.attesterClient.CrosslinkCommitteesAtSlot(ctx, req)
	if err != nil {
		log.Errorf("Could not fetch crosslink committees at slot %d: %v", err)
		return
	}
	// Set the attestation data's shard as the shard associated with the validator's
	// committee as retrieved by CrosslinkCommitteesAtSlot.
	attData.Shard = resp.Shard

	// Fetch other necessary information from the beacon node in order to attest
	// including the justified epoch, epoch boundary information, and more.
	req := &pb.AttestationInfoRequest{
		Slot: slot,
	}
	resp, err := v.attesterClient.AttestationInfoAtSlot(ctx, req)
	if err != nil {
		log.Errorf("Could not fetch necessary info to produce attestation at slot %d: %v", err)
		return
	}
	// Set the attestation data's beacon block root = hash_tree_root(head) where head
	// is the validator's view of the head block of the beacon chain during the slot.
	attData.BeaconBlockRootHash32 = resp.BeaconBlockRootHash32[:]
	// Set the attestation data's epoch boundary root = hash_tree_root(epoch_boundary)
	// where epoch_boundary is the block at the most recent epoch boundary in the
	// chain defined by head -- i.e. the BeaconBlock where block.slot == get_epoch_start_slot(head.slot).
	// On the server side, this is fetched by calling get_block_root(state, get_epoch_start_slot(head.slot)).
	attData.EpochBoundaryRootHash32 = resp.EpochBoundaryRootHash32[:]
	// Set the attestation data's justified epoch = state.justified_epoch where state
	// is the beacon state at the head.
	attData.JustifiedSlot = resp.JustifiedEpoch
	// Set the attestation data's justified block root = hash_tree_root(justified_block) where
	// justified_block is the block at state.justified_epoch in the chain defined by head.
	// On the server side, this is fetched by calling get_block_root(state, justified_epoch).
	attData.JustifiedBlockRootHash32 = resp.JustifiedBlockRootHash32[:]

	// The validator now creates an Attestation object using the AttestationData as
	// set in the code above after all properties have been set.
	attestation := &pbp2p.Attestation{
		Data: attData,
	}
	_ = attestation
}
