package blocks

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
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
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	var err error
	for idx, attestation := range body.Attestations {
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
//    assert data.index < get_committee_count_at_slot(state, data.slot)
//    assert data.target.epoch in (get_previous_epoch(state), get_current_epoch(state))
//    assert data.target.epoch == compute_epoch_at_slot(data.slot)
//    assert data.slot + MIN_ATTESTATION_INCLUSION_DELAY <= state.slot <= data.slot + SLOTS_PER_EPOCH
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
	beaconState *stateTrie.BeaconState,
	att *ethpb.Attestation,
) (*stateTrie.BeaconState, error) {
	beaconState, err := ProcessAttestationNoVerify(ctx, beaconState, att)
	if err != nil {
		return nil, err
	}
	return beaconState, VerifyAttestation(ctx, beaconState, att)
}

// ProcessAttestationsNoVerify applies processing operations to a block's inner attestation
// records. The only difference would be that the attestation signature would not be verified.
func ProcessAttestationsNoVerify(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	var err error
	for idx, attestation := range body.Attestations {
		beaconState, err = ProcessAttestationNoVerify(ctx, beaconState, attestation)
		if err != nil {
			return nil, errors.Wrapf(err, "could not verify attestation at index %d in block", idx)
		}
	}
	return beaconState, nil
}

// ProcessAttestationNoVerify processes the attestation without verifying the attestation signature. This
// method is used to validate attestations whose signatures have already been verified.
func ProcessAttestationNoVerify(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	att *ethpb.Attestation,
) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "core.ProcessAttestationNoVerify")
	defer span.End()

	if att == nil || att.Data == nil || att.Data.Target == nil {
		return nil, errors.New("nil attestation data target")
	}

	currEpoch := helpers.SlotToEpoch(beaconState.Slot())
	var prevEpoch uint64
	if currEpoch == 0 {
		prevEpoch = 0
	} else {
		prevEpoch = currEpoch - 1
	}
	data := att.Data
	if data.Target.Epoch != prevEpoch && data.Target.Epoch != currEpoch {
		return nil, fmt.Errorf(
			"expected target epoch (%d) to be the previous epoch (%d) or the current epoch (%d)",
			data.Target.Epoch,
			prevEpoch,
			currEpoch,
		)
	}
	if helpers.SlotToEpoch(data.Slot) != data.Target.Epoch {
		return nil, fmt.Errorf("data slot is not in the same epoch as target %d != %d", helpers.SlotToEpoch(data.Slot), data.Target.Epoch)
	}

	s := att.Data.Slot
	minInclusionCheck := s+params.BeaconConfig().MinAttestationInclusionDelay <= beaconState.Slot()
	epochInclusionCheck := beaconState.Slot() <= s+params.BeaconConfig().SlotsPerEpoch
	if !minInclusionCheck {
		return nil, fmt.Errorf(
			"attestation slot %d + inclusion delay %d > state slot %d",
			s,
			params.BeaconConfig().MinAttestationInclusionDelay,
			beaconState.Slot(),
		)
	}
	if !epochInclusionCheck {
		return nil, fmt.Errorf(
			"state slot %d > attestation slot %d + SLOTS_PER_EPOCH %d",
			beaconState.Slot(),
			s,
			params.BeaconConfig().SlotsPerEpoch,
		)
	}

	if err := helpers.VerifyAttestationBitfieldLengths(beaconState, att); err != nil {
		return nil, errors.Wrap(err, "could not verify attestation bitfields")
	}

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

	var ffgSourceEpoch uint64
	var ffgSourceRoot []byte
	var ffgTargetEpoch uint64
	if data.Target.Epoch == currEpoch {
		ffgSourceEpoch = beaconState.CurrentJustifiedCheckpoint().Epoch
		ffgSourceRoot = beaconState.CurrentJustifiedCheckpoint().Root
		ffgTargetEpoch = currEpoch
		if err := beaconState.AppendCurrentEpochAttestations(pendingAtt); err != nil {
			return nil, err
		}
	} else {
		ffgSourceEpoch = beaconState.PreviousJustifiedCheckpoint().Epoch
		ffgSourceRoot = beaconState.PreviousJustifiedCheckpoint().Root
		ffgTargetEpoch = prevEpoch
		if err := beaconState.AppendPreviousEpochAttestations(pendingAtt); err != nil {
			return nil, err
		}
	}
	if data.Source.Epoch != ffgSourceEpoch {
		return nil, fmt.Errorf("expected source epoch %d, received %d", ffgSourceEpoch, data.Source.Epoch)
	}
	if !bytes.Equal(data.Source.Root, ffgSourceRoot) {
		return nil, fmt.Errorf("expected source root %#x, received %#x", ffgSourceRoot, data.Source.Root)
	}
	if data.Target.Epoch != ffgTargetEpoch {
		return nil, fmt.Errorf("expected target epoch %d, received %d", ffgTargetEpoch, data.Target.Epoch)
	}

	return beaconState, nil
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
func VerifyIndexedAttestation(ctx context.Context, beaconState *stateTrie.BeaconState, indexedAtt *ethpb.IndexedAttestation) error {
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
	pubkeys := []bls.PublicKey{}
	for i := 0; i < len(indices); i++ {
		pubkeyAtIdx := beaconState.PubkeyAtIndex(indices[i])
		pk, err := bls.PublicKeyFromBytes(pubkeyAtIdx[:])
		if err != nil {
			return errors.Wrap(err, "could not deserialize validator public key")
		}
		pubkeys = append(pubkeys, pk)
	}
	return attestationutil.VerifyIndexedAttestationSig(ctx, indexedAtt, pubkeys, domain)
}

// VerifyAttestation converts and attestation into an indexed attestation and verifies
// the signature in that attestation.
func VerifyAttestation(ctx context.Context, beaconState *stateTrie.BeaconState, att *ethpb.Attestation) error {
	if att == nil || att.Data == nil {
		return fmt.Errorf("nil or missing attestation data: %v", att)
	}
	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		return err
	}
	indexedAtt := attestationutil.ConvertToIndexed(ctx, att, committee)
	return VerifyIndexedAttestation(ctx, beaconState, indexedAtt)
}

// VerifyAttestations will verify the signatures of the provided attestations. This method performs
// a single BLS verification call to verify the signatures of all of the provided attestations. All
// of the provided attestations must have valid signatures or this method will return an error.
// This method does not determine which attestation signature is invalid, only that one or more
// attestation signatures were not valid.
func VerifyAttestations(ctx context.Context, beaconState *stateTrie.BeaconState, atts []*ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "core.VerifyAttestations")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("attestations", int64(len(atts))))

	if len(atts) == 0 {
		return nil
	}

	fork := beaconState.Fork()
	gvr := beaconState.GenesisValidatorRoot()
	dt := params.BeaconConfig().DomainBeaconAttester

	// Split attestations by fork. Note: the signature domain will differ based on the fork.
	var preForkAtts []*ethpb.Attestation
	var postForkAtts []*ethpb.Attestation
	for _, a := range atts {
		if helpers.SlotToEpoch(a.Data.Slot) < fork.Epoch {
			preForkAtts = append(preForkAtts, a)
		} else {
			postForkAtts = append(postForkAtts, a)
		}
	}

	// Check attestations from before the fork.
	if fork.Epoch > 0 { // Check to prevent underflow.
		prevDomain, err := helpers.Domain(fork, fork.Epoch-1, dt, gvr)
		if err != nil {
			return err
		}
		if err := verifyAttestationsWithDomain(ctx, beaconState, preForkAtts, prevDomain); err != nil {
			return err
		}
	} else if len(preForkAtts) > 0 {
		// This is a sanity check that preForkAtts were not ignored when fork.Epoch == 0. This
		// condition is not possible, but it doesn't hurt to check anyway.
		return errors.New("some attestations were not verified from previous fork before genesis")
	}

	// Then check attestations from after the fork.
	currDomain, err := helpers.Domain(fork, fork.Epoch, dt, gvr)
	if err != nil {
		return err
	}

	return verifyAttestationsWithDomain(ctx, beaconState, postForkAtts, currDomain)
}

// Inner method to verify attestations. This abstraction allows for the domain to be provided as an
// argument.
func verifyAttestationsWithDomain(ctx context.Context, beaconState *stateTrie.BeaconState, atts []*ethpb.Attestation, domain []byte) error {
	if len(atts) == 0 {
		return nil
	}
	set, err := createAttestationSignatureSet(ctx, beaconState, atts, domain)
	if err != nil {
		return err
	}
	verify, err := set.Verify()
	if err != nil {
		return errors.Errorf("got error in multiple verification: %v", err)
	}
	if !verify {
		return errors.New("one or more attestation signatures did not verify")
	}
	return nil
}
