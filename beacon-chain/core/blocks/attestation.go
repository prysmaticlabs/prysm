package blocks

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// ProcessAttestations applies processing operations to a block's inner attestation
// records.
func ProcessAttestations(
	ctx context.Context,
	beaconState iface.BeaconState,
	b *ethpb.SignedBeaconBlock,
) (iface.BeaconState, error) {
	if err := helpers.VerifyNilBeaconBlock(b); err != nil {
		return nil, err
	}

	var err error
	for idx, attestation := range b.Block.Body.Attestations {
		beaconState, err = ProcessAttestation(ctx, beaconState, attestation)
		if err != nil {
			return nil, errors.Wrapf(err, "could not verify attestation at index %d in block", idx)
		}
	}
	return beaconState, nil
}

// ProcessAttestation verifies an input attestation can pass through processing using the given beacon state.
//
// Spec pseudocode definition:
//  def process_attestation(state: BeaconState, attestation: Attestation) -> None:
//    data = attestation.data
//    assert data.target.epoch in (get_previous_epoch(state), get_current_epoch(state))
//    assert data.target.epoch == compute_epoch_at_slot(data.slot)
//    assert data.slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot <= data.slot + SLOTS_PER_EPOCH
//    assert data.index < get_committee_count_per_slot(state, data.target.epoch)
//
//    committee = get_beacon_committee(state, data.slot, data.index)
//    assert len(attestation.aggregation_bits) == len(committee)
//
//    pending_attestation = PendingAttestation(
//        data=data,
//        aggregation_bits=attestation.aggregation_bits,
//        inclusion_delay=state.slot - data.slot,
//        proposer_index=get_beacon_proposer_index(state),
//    )
//
//    if data.target.epoch == get_current_epoch(state):
//        assert data.source == state.current_justified_checkpoint
//        state.current_epoch_attestations.append(pending_attestation)
//    else:
//        assert data.source == state.previous_justified_checkpoint
//        state.previous_epoch_attestations.append(pending_attestation)
//
//    # Check signature
//    assert is_valid_indexed_attestation(state, get_indexed_attestation(state, attestation))
func ProcessAttestation(
	ctx context.Context,
	beaconState iface.BeaconState,
	att *ethpb.Attestation,
) (iface.BeaconState, error) {
	beaconState, err := ProcessAttestationNoVerifySignature(ctx, beaconState, att)
	if err != nil {
		return nil, err
	}
	return beaconState, VerifyAttestationSignature(ctx, beaconState, att)
}

// ProcessAttestationsNoVerifySignature applies processing operations to a block's inner attestation
// records. The only difference would be that the attestation signature would not be verified.
func ProcessAttestationsNoVerifySignature(
	ctx context.Context,
	beaconState iface.BeaconState,
	b *ethpb.SignedBeaconBlock,
) (iface.BeaconState, error) {
	if err := helpers.VerifyNilBeaconBlock(b); err != nil {
		return nil, err
	}
	body := b.Block.Body
	var err error
	for idx, attestation := range body.Attestations {
		beaconState, err = ProcessAttestationNoVerifySignature(ctx, beaconState, attestation)
		if err != nil {
			return nil, errors.Wrapf(err, "could not verify attestation at index %d in block", idx)
		}
	}
	return beaconState, nil
}

// VerifyAttestationNoVerifySignature verifies the attestation without verifying the attestation signature. This is
// used before processing attestation with the beacon state.
func VerifyAttestationNoVerifySignature(
	ctx context.Context,
	beaconState iface.ReadOnlyBeaconState,
	att *ethpb.Attestation,
) error {
	ctx, span := trace.StartSpan(ctx, "core.VerifyAttestationNoVerifySignature")
	defer span.End()

	if err := helpers.ValidateNilAttestation(att); err != nil {
		return err
	}
	currEpoch := helpers.CurrentEpoch(beaconState)
	prevEpoch := helpers.PrevEpoch(beaconState)
	data := att.Data
	if data.Target.Epoch != prevEpoch && data.Target.Epoch != currEpoch {
		return fmt.Errorf(
			"expected target epoch (%d) to be the previous epoch (%d) or the current epoch (%d)",
			data.Target.Epoch,
			prevEpoch,
			currEpoch,
		)
	}

	if data.Target.Epoch == currEpoch {
		if !beaconState.MatchCurrentJustifiedCheckpoint(data.Source) {
			return errors.New("source check point not equal to current justified checkpoint")
		}
	} else {
		if !beaconState.MatchPreviousJustifiedCheckpoint(data.Source) {
			return errors.New("source check point not equal to previous justified checkpoint")
		}
	}

	if err := helpers.ValidateSlotTargetEpoch(att.Data); err != nil {
		return err
	}

	s := att.Data.Slot
	minInclusionCheck := s+params.BeaconConfig().MinAttestationInclusionDelay <= beaconState.Slot()
	epochInclusionCheck := beaconState.Slot() <= s+params.BeaconConfig().SlotsPerEpoch
	if !minInclusionCheck {
		return fmt.Errorf(
			"attestation slot %d + inclusion delay %d > state slot %d",
			s,
			params.BeaconConfig().MinAttestationInclusionDelay,
			beaconState.Slot(),
		)
	}
	if !epochInclusionCheck {
		return fmt.Errorf(
			"state slot %d > attestation slot %d + SLOTS_PER_EPOCH %d",
			beaconState.Slot(),
			s,
			params.BeaconConfig().SlotsPerEpoch,
		)
	}
	activeValidatorCount, err := helpers.ActiveValidatorCount(beaconState, att.Data.Target.Epoch)
	if err != nil {
		return err
	}
	c := helpers.SlotCommitteeCount(activeValidatorCount)
	if uint64(att.Data.CommitteeIndex) >= c {
		return fmt.Errorf("committee index %d >= committee count %d", att.Data.CommitteeIndex, c)
	}

	if err := helpers.VerifyAttestationBitfieldLengths(beaconState, att); err != nil {
		return errors.Wrap(err, "could not verify attestation bitfields")
	}

	// Verify attesting indices are correct.
	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		return err
	}
	indexedAtt, err := attestationutil.ConvertToIndexed(ctx, att, committee)
	if err != nil {
		return err
	}

	return attestationutil.IsValidAttestationIndices(ctx, indexedAtt)
}

// ProcessAttestationNoVerifySignature processes the attestation without verifying the attestation signature. This
// method is used to validate attestations whose signatures have already been verified.
func ProcessAttestationNoVerifySignature(
	ctx context.Context,
	beaconState iface.BeaconState,
	att *ethpb.Attestation,
) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "core.ProcessAttestationNoVerifySignature")
	defer span.End()

	if err := VerifyAttestationNoVerifySignature(ctx, beaconState, att); err != nil {
		return nil, err
	}

	currEpoch := helpers.CurrentEpoch(beaconState)
	data := att.Data
	s := att.Data.Slot
	proposerIndex, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return nil, err
	}
	pendingAtt := &pb.PendingAttestation{
		Data:            data,
		AggregationBits: att.AggregationBits,
		InclusionDelay:  beaconState.Slot() - s,
		ProposerIndex:   proposerIndex,
	}

	if data.Target.Epoch == currEpoch {
		if err := beaconState.AppendCurrentEpochAttestations(pendingAtt); err != nil {
			return nil, err
		}
	} else {
		if err := beaconState.AppendPreviousEpochAttestations(pendingAtt); err != nil {
			return nil, err
		}
	}

	return beaconState, nil
}

// VerifyAttestationSignature converts and attestation into an indexed attestation and verifies
// the signature in that attestation.
func VerifyAttestationSignature(ctx context.Context, beaconState iface.ReadOnlyBeaconState, att *ethpb.Attestation) error {
	if err := helpers.ValidateNilAttestation(att); err != nil {
		return err
	}
	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		return err
	}
	indexedAtt, err := attestationutil.ConvertToIndexed(ctx, att, committee)
	if err != nil {
		return err
	}
	return VerifyIndexedAttestation(ctx, beaconState, indexedAtt)
}

// VerifyIndexedAttestation determines the validity of an indexed attestation.
//
// Spec pseudocode definition:
//  def is_valid_indexed_attestation(state: BeaconState, indexed_attestation: IndexedAttestation) -> bool:
//    """
//    Check if ``indexed_attestation`` is not empty, has sorted and unique indices and has a valid aggregate signature.
//    """
//    # Verify indices are sorted and unique
//    indices = indexed_attestation.attesting_indices
//    if len(indices) == 0 or not indices == sorted(set(indices)):
//        return False
//    # Verify aggregate signature
//    pubkeys = [state.validators[i].pubkey for i in indices]
//    domain = get_domain(state, DOMAIN_BEACON_ATTESTER, indexed_attestation.data.target.epoch)
//    signing_root = compute_signing_root(indexed_attestation.data, domain)
//    return bls.FastAggregateVerify(pubkeys, signing_root, indexed_attestation.signature)
func VerifyIndexedAttestation(ctx context.Context, beaconState iface.ReadOnlyBeaconState, indexedAtt *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "core.VerifyIndexedAttestation")
	defer span.End()

	if err := attestationutil.IsValidAttestationIndices(ctx, indexedAtt); err != nil {
		return err
	}
	domain, err := helpers.Domain(beaconState.Fork(), indexedAtt.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		return err
	}
	indices := indexedAtt.AttestingIndices
	var pubkeys []bls.PublicKey
	for i := 0; i < len(indices); i++ {
		pubkeyAtIdx := beaconState.PubkeyAtIndex(types.ValidatorIndex(indices[i]))
		pk, err := bls.PublicKeyFromBytes(pubkeyAtIdx[:])
		if err != nil {
			return errors.Wrap(err, "could not deserialize validator public key")
		}
		pubkeys = append(pubkeys, pk)
	}
	return attestationutil.VerifyIndexedAttestationSig(ctx, indexedAtt, pubkeys, domain)
}
