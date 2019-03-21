package client

import (
	"context"
	"fmt"
	"time"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var delay = params.BeaconConfig().SecondsPerSlot / 2

// AttestToBlockHead completes the validator client's attester responsibility at a given slot.
// It fetches the latest beacon block head along with the latest canonical beacon state
// information in order to sign the block and include information about the validator's
// participation in voting on the block.
func (v *validator) AttestToBlockHead(ctx context.Context, slot uint64) {
	ctx, span := trace.StartSpan(ctx, "validator.AttestToBlockHead")
	defer span.End()

	v.waitToSlotMidpoint(ctx, slot)

	// First the validator should construct attestation_data, an AttestationData
	// object based upon the state at the assigned slot.
	attData := &pbp2p.AttestationData{
		Slot:                    slot,
		CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:], // Stub for Phase 0.
	}
	// We fetch the validator index as it is necessary to generate the aggregation
	// bitfield of the attestation itself.
	pubKey := v.key.PublicKey.Marshal()
	idxReq := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	validatorIndexRes, err := v.validatorClient.ValidatorIndex(ctx, idxReq)
	if err != nil {
		log.Errorf("Could not fetch validator index: %v", err)
		return
	}
	req := &pb.ValidatorEpochAssignmentsRequest{
		EpochStart: slot,
		PublicKey:  pubKey,
	}
	resp, err := v.validatorClient.CommitteeAssignment(ctx, req)
	if err != nil {
		log.Errorf("Could not fetch crosslink committees at slot %d: %v",
			slot-params.BeaconConfig().GenesisSlot, err)
		return
	}
	// Set the attestation data's shard as the shard associated with the validator's
	// committee as retrieved by CrosslinkCommitteesAtSlot.
	attData.Shard = resp.Shard

	// Fetch other necessary information from the beacon node in order to attest
	// including the justified epoch, epoch boundary information, and more.
	infoReq := &pb.AttestationDataRequest{
		Slot:  slot,
		Shard: resp.Shard,
	}
	infoRes, err := v.attesterClient.AttestationDataAtSlot(ctx, infoReq)
	if err != nil {
		log.Errorf("Could not fetch necessary info to produce attestation at slot %d: %v",
			slot-params.BeaconConfig().GenesisSlot, err)
		return
	}
	// Set the attestation data's beacon block root = hash_tree_root(head) where head
	// is the validator's view of the head block of the beacon chain during the slot.
	attData.BeaconBlockRootHash32 = infoRes.BeaconBlockRootHash32
	// Set the attestation data's epoch boundary root = hash_tree_root(epoch_boundary)
	// where epoch_boundary is the block at the most recent epoch boundary in the
	// chain defined by head -- i.e. the BeaconBlock where block.slot == get_epoch_start_slot(slot_to_epoch(head.slot)).
	attData.EpochBoundaryRootHash32 = infoRes.EpochBoundaryRootHash32
	// Set the attestation data's latest crosslink root = state.latest_crosslinks[shard].shard_block_root
	// where state is the beacon state at head and shard is the validator's assigned shard.
	attData.LatestCrosslink = infoRes.LatestCrosslink
	// Set the attestation data's justified epoch = state.justified_epoch where state
	// is the beacon state at the head.
	attData.JustifiedEpoch = infoRes.JustifiedEpoch
	// Set the attestation data's justified block root = hash_tree_root(justified_block) where
	// justified_block is the block at state.justified_epoch in the chain defined by head.
	// On the server side, this is fetched by calling get_block_root(state, justified_epoch).
	attData.JustifiedBlockRootHash32 = infoRes.JustifiedBlockRootHash32

	// The validator now creates an Attestation object using the AttestationData as
	// set in the code above after all properties have been set.
	attestation := &pbp2p.Attestation{
		Data: attData,
	}

	// We set the custody bitfield to an slice of zero values as a stub for phase 0
	// of length len(committee)+7 // 8.
	attestation.CustodyBitfield = make([]byte, (len(resp.Committee)+7)/8)

	// Find the index in committee to be used for
	// the aggregation bitfield
	var indexInCommittee int
	for i, vIndex := range resp.Committee {
		if vIndex == validatorIndexRes.Index {
			indexInCommittee = i
			break
		}
	}

	aggregationBitfield := bitutil.SetBitfield(indexInCommittee)
	attestation.AggregationBitfield = aggregationBitfield

	// TODO(#1366): Use BLS to generate an aggregate signature.
	attestation.AggregateSignature = []byte("signed")

	log.WithField(
		"blockRoot", fmt.Sprintf("%#x", attData.BeaconBlockRootHash32),
	).Info("Current beacon chain head block")
	log.WithFields(logrus.Fields{
		"justifiedEpoch": attData.JustifiedEpoch - params.BeaconConfig().GenesisEpoch,
		"shard":          attData.Shard,
		"slot":           slot - params.BeaconConfig().GenesisSlot,
	}).Info("Attesting to beacon chain head...")

	log.Debugf("Produced attestation: %v", attestation)
	attResp, err := v.attesterClient.AttestHead(ctx, attestation)
	if err != nil {
		log.Errorf("Could not submit attestation to beacon node: %v", err)
		return
	}
	log.WithFields(logrus.Fields{
		"attestationHash": fmt.Sprintf("%#x", attResp.AttestationHash),
		"shard":           attData.Shard,
		"slot":            slot - params.BeaconConfig().GenesisSlot,
	}).Info("Beacon node processed attestation successfully")
}

// waitToSlotMidpoint waits until halfway through the current slot period
// such that any blocks from this slot have time to reach the beacon node
// before creating the attestation.
func (v *validator) waitToSlotMidpoint(ctx context.Context, slot uint64) {
	_, span := trace.StartSpan(ctx, "validator.waitToSlotMidpoint")
	defer span.End()

	duration := time.Duration(slot*params.BeaconConfig().SecondsPerSlot+delay) * time.Second
	timeToBroadcast := time.Unix(int64(v.genesisTime), 0).Add(duration)

	time.Sleep(time.Until(timeToBroadcast))
}
