package client

import (
	"bytes"
	"context"
	"fmt"
	"time"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var delay = params.BeaconConfig().SecondsPerSlot / 2

// AttestToBlockHead completes the validator client's attester responsibility at a given slot.
// It fetches the latest beacon block head along with the latest canonical beacon state
// information in order to sign the block and include information about the validator's
// participation in voting on the block.
func (v *validator) AttestToBlockHead(ctx context.Context, slot uint64, idx string) {
	ctx, span := trace.StartSpan(ctx, "validator.AttestToBlockHead")
	defer span.End()
	span.AddAttributes(
		trace.StringAttribute("validator", fmt.Sprintf("%#x", v.keys[idx].PublicKey.Marshal())),
	)
	truncatedPk := idx
	if len(idx) > 12 {
		truncatedPk = idx[:12]
	}
	log.WithField("validator", truncatedPk).Info("Performing a beacon block attestation...")
	v.waitToSlotMidpoint(ctx, slot)

	// First the validator should construct attestation_data, an AttestationData
	// object based upon the state at the assigned slot.
	attData := &pbp2p.AttestationData{
		CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:], // Stub for Phase 0.
	}
	// We fetch the validator index as it is necessary to generate the aggregation
	// bitfield of the attestation itself.
	pubKey := v.keys[idx].PublicKey.Marshal()
	var assignment *pb.CommitteeAssignmentResponse_CommitteeAssignment
	if v.assignments == nil {
		log.Errorf("No assignments for validators")
		return
	}
	for _, amnt := range v.assignments.Assignment {
		if bytes.Equal(pubKey, amnt.PublicKey) {
			assignment = amnt
		}
	}
	idxReq := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	validatorIndexRes, err := v.validatorClient.ValidatorIndex(ctx, idxReq)
	if err != nil {
		log.Errorf("Could not fetch validator index: %v", err)
		return
	}
	// Set the attestation data's shard as the shard associated with the validator's
	// committee as retrieved by CrosslinkCommitteesAtSlot.
	attData.Shard = assignment.Shard

	// Fetch other necessary information from the beacon node in order to attest
	// including the justified epoch, epoch boundary information, and more.
	infoReq := &pb.AttestationDataRequest{
		Slot:  slot,
		Shard: assignment.Shard,
	}
	infoRes, err := v.attesterClient.AttestationDataAtSlot(ctx, infoReq)
	if err != nil {
		log.Errorf("Could not fetch necessary info to produce attestation at slot %d: %v",
			slot-params.BeaconConfig().GenesisSlot, err)
		return
	}

	committeeLength := mathutil.CeilDiv8(len(assignment.Committee))

	// Set the attestation data's slot to head_state.slot where the slot
	// is the canonical head of the beacon chain.
	attData.Slot = infoRes.HeadSlot
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
	attestation.CustodyBitfield = make([]byte, committeeLength)

	// Find the index in committee to be used for
	// the aggregation bitfield
	var indexInCommittee int
	for i, vIndex := range assignment.Committee {
		if vIndex == validatorIndexRes.Index {
			indexInCommittee = i
			break
		}
	}

	aggregationBitfield := bitutil.SetBitfield(indexInCommittee, committeeLength)
	attestation.AggregationBitfield = aggregationBitfield

	// TODO(#1366): Use BLS to generate an aggregate signature.
	attestation.AggregateSignature = []byte("signed")

	log.WithFields(logrus.Fields{
		"shard":     attData.Shard,
		"slot":      slot - params.BeaconConfig().GenesisSlot,
		"validator": truncatedPk,
	}).Info("Attesting to beacon chain head...")

	attResp, err := v.attesterClient.AttestHead(ctx, attestation)
	if err != nil {
		log.Errorf("Could not submit attestation to beacon node: %v", err)
		return
	}
	log.WithFields(logrus.Fields{
		"headRoot":  fmt.Sprintf("%#x", bytesutil.Trunc(attData.BeaconBlockRootHash32)),
		"slot":      attData.Slot - params.BeaconConfig().GenesisSlot,
		"shard":     attData.Shard,
		"validator": truncatedPk,
	}).Info("Attested latest head")
	span.AddAttributes(
		trace.Int64Attribute("slot", int64(slot-params.BeaconConfig().GenesisSlot)),
		trace.StringAttribute("attestationHash", fmt.Sprintf("%#x", attResp.AttestationHash)),
		trace.Int64Attribute("shard", int64(attData.Shard)),
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", attestation.Data.BeaconBlockRootHash32)),
		trace.Int64Attribute("justifiedEpoch", int64(attData.JustifiedEpoch-params.BeaconConfig().GenesisEpoch)),
		trace.StringAttribute("bitfield", fmt.Sprintf("%#x", aggregationBitfield)),
	)
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
