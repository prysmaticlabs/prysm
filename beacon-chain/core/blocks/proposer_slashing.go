package blocks

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/protobuf/proto"
)

type slashValidatorFunc func(state.BeaconState, types.ValidatorIndex) (state.BeaconState, error)

// ProcessProposerSlashings is one of the operations performed
// on each processed beacon block to slash proposers based on
// slashing conditions if any slashable events occurred.
//
// Spec pseudocode definition:
//   def process_proposer_slashing(state: BeaconState, proposer_slashing: ProposerSlashing) -> None:
//    header_1 = proposer_slashing.signed_header_1.message
//    header_2 = proposer_slashing.signed_header_2.message
//
//    # Verify header slots match
//    assert header_1.slot == header_2.slot
//    # Verify header proposer indices match
//    assert header_1.proposer_index == header_2.proposer_index
//    # Verify the headers are different
//    assert header_1 != header_2
//    # Verify the proposer is slashable
//    proposer = state.validators[header_1.proposer_index]
//    assert is_slashable_validator(proposer, get_current_epoch(state))
//    # Verify signatures
//    for signed_header in (proposer_slashing.signed_header_1, proposer_slashing.signed_header_2):
//        domain = get_domain(state, DOMAIN_BEACON_PROPOSER, compute_epoch_at_slot(signed_header.message.slot))
//        signing_root = compute_signing_root(signed_header.message, domain)
//        assert bls.Verify(proposer.pubkey, signing_root, signed_header.signature)
//
//    slash_validator(state, header_1.proposer_index)
func ProcessProposerSlashings(
	_ context.Context,
	beaconState state.BeaconState,
	slashings []*ethpb.ProposerSlashing,
	slashFunc slashValidatorFunc,
) (state.BeaconState, error) {
	var err error
	for idx, slashing := range slashings {
		if slashing == nil {
			return nil, errors.New("nil proposer slashings in block body")
		}
		if err = VerifyProposerSlashing(beaconState, slashing); err != nil {
			return nil, errors.Wrapf(err, "could not verify proposer slashing %d", idx)
		}
		beaconState, err = slashFunc(beaconState, slashing.Header_1.Header.ProposerIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "could not slash proposer index %d", slashing.Header_1.Header.ProposerIndex)
		}
	}
	return beaconState, nil
}

// VerifyProposerSlashing verifies that the data provided from slashing is valid.
func VerifyProposerSlashing(
	beaconState state.BeaconState,
	slashing *ethpb.ProposerSlashing,
) error {
	if slashing.Header_1 == nil || slashing.Header_1.Header == nil || slashing.Header_2 == nil || slashing.Header_2.Header == nil {
		return errors.New("nil header cannot be verified")
	}
	hSlot := slashing.Header_1.Header.Slot
	if hSlot != slashing.Header_2.Header.Slot {
		return fmt.Errorf("mismatched header slots, received %d == %d", slashing.Header_1.Header.Slot, slashing.Header_2.Header.Slot)
	}
	pIdx := slashing.Header_1.Header.ProposerIndex
	if pIdx != slashing.Header_2.Header.ProposerIndex {
		return fmt.Errorf("mismatched indices, received %d == %d", slashing.Header_1.Header.ProposerIndex, slashing.Header_2.Header.ProposerIndex)
	}
	if proto.Equal(slashing.Header_1.Header, slashing.Header_2.Header) {
		return errors.New("expected slashing headers to differ")
	}
	proposer, err := beaconState.ValidatorAtIndexReadOnly(slashing.Header_1.Header.ProposerIndex)
	if err != nil {
		return err
	}
	if !helpers.IsSlashableValidatorUsingTrie(proposer, helpers.CurrentEpoch(beaconState)) {
		return fmt.Errorf("validator with key %#x is not slashable", proposer.PublicKey())
	}
	headers := []*ethpb.SignedBeaconBlockHeader{slashing.Header_1, slashing.Header_2}
	for _, header := range headers {
		if err := helpers.ComputeDomainVerifySigningRoot(beaconState, pIdx, helpers.SlotToEpoch(hSlot),
			header.Header, params.BeaconConfig().DomainBeaconProposer, header.Signature); err != nil {
			return errors.Wrap(err, "could not verify beacon block header")
		}
	}
	return nil
}
