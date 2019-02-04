package client

import (
	"context"
	ptypes "github.com/gogo/protobuf/types"

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
		Slot: slot,
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

	// Set the attestation data's beacon block root = hash_tree_root(head) where head
	// is the validator's view of the head block of the beacon chain during the slot.
	headBlock, err := v.beaconClient.CanonicalHead(ctx, &ptypes.Empty{})
	if err != nil {
		log.Errorf("Failed to fetch CanonicalHead: %v", err)
		return
	}
	headBlockRoot, err := ssz.TreeHash(headBlock)
	if err != nil {
		log.Errorf("Failed to tree hash current beacon chain head: %v", err)
		return
	}
	attData.BeaconBlockRootHash32 = headBlockRoot[:]
}
