package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// validateAggregateAndProof validates the aggregated signature and its proof are valid before forwarding to the
// network and downstream services.
func (r *RegularSync) validateAggregateAndProof(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) (bool, error) {
	// To process the following it requires the recent blocks to be present in the database, so we'll skip
	// validating or processing aggregated attestations until fully synced.
	if r.initialSync.Syncing() {
		return false, nil
	}

	m := msg.(*pb.AggregateAndProof)
	attSlot := m.Aggregate.Data.Slot

	// Verify aggregate attestation has not already been seen via aggregate gossip, within a block, or through the creation locally.
	// TODO(3835): Blocked by operation pool redesign


	// Verify the block being voted for passes validation. The block should have passed validation if it's in the DB.
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(m.Aggregate.Data.BeaconBlockRoot)) {
		return false, errPointsToBlockNotInDatabase
	}

	// Verify attestation slot is within the last ATTESTATION_PROPAGATION_SLOT_RANGE slots.
	headState, err := r.chain.HeadState(ctx)
	if err != nil {
		return false, err
	}
	currentSlot := (uint64(time.Now().Unix()) - headState.GenesisTime) / params.BeaconConfig().SecondsPerSlot
	if attSlot > currentSlot || currentSlot > attSlot + params.BeaconConfig().AttestationPropagationSlotRange {
		return false, fmt.Errorf("attestation slot out of range %d < %d < %d",
			attSlot, currentSlot, attSlot + params.BeaconConfig().AttestationPropagationSlotRange)
	}

	// Verify validator index is within the aggregate's committee.
	currentState, err := state.ProcessSlots(ctx, headState, attSlot)
	if err != nil {
		return false, err
	}
	attestingIndices, err := helpers.AttestingIndices(currentState, m.Aggregate.Data, m.Aggregate.AggregationBits)
	var withinCommittee bool
	for _, i := range attestingIndices {
		if m.AggregatorIndex == i {
			withinCommittee = true
			break
		}
	}
	if !withinCommittee {
		return false, fmt.Errorf("validator index %d is not within the committee %v",
			m.AggregatorIndex, attestingIndices)
	}

	// Verify selection proof selects the validator as an aggregator for the slot.
	slotSig, err := bls.SignatureFromBytes(m.SelectionProof)
	if err != nil {
		return false, err
	}
	aggregator, err := helpers.IsAggregator(currentState, attSlot, m.Aggregate.Data.CommitteeIndex, slotSig)
	if err != nil {
		return false, err
	}
	if !aggregator {
		return false, fmt.Errorf("validator index %d is not an aggregator for slot %d",
			m.AggregatorIndex, attSlot)
	}

	// Verify selection proof is a valid signature
	domain := helpers.Domain(currentState.Fork, helpers.SlotToEpoch(attSlot), params.BeaconConfig().DomainBeaconAttester)
	slotMsg, err := ssz.HashTreeRoot(attSlot)
	if err != nil {
		return false, err
	}
	pubKey, err := bls.PublicKeyFromBytes(currentState.Validators[m.AggregatorIndex].PublicKey)
	if err != nil {
		return false, err
	}
	if !slotSig.Verify(slotMsg[:], pubKey, domain) {
		return false, err
	}

	// Verify aggregated attestation has a valid signature
	if err := blocks.VerifyAttestation(ctx, currentState, m.Aggregate); err != nil {
		return false, err
	}

	return true, nil
}
