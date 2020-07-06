package blocks

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessProposerSlashings is one of the operations performed
// on each processed beacon block to slash proposers based on
// slashing conditions if any slashable events occurred.
//
// Spec pseudocode definition:
//   def process_proposer_slashing(state: BeaconState, proposer_slashing: ProposerSlashing) -> None:
//    """
//    Process ``ProposerSlashing`` operation.
//    """
//    proposer = state.validator_registry[proposer_slashing.proposer_index]
//    # Verify slots match
//    assert proposer_slashing.header_1.slot == proposer_slashing.header_2.slot
//    # But the headers are different
//    assert proposer_slashing.header_1 != proposer_slashing.header_2
//    # Check proposer is slashable
//    assert is_slashable_validator(proposer, get_current_epoch(state))
//    # Signatures are valid
//    for header in (proposer_slashing.header_1, proposer_slashing.header_2):
//        domain = get_domain(state, DOMAIN_BEACON_PROPOSER, slot_to_epoch(header.slot))
//        assert bls_verify(proposer.pubkey, signing_root(header), header.signature, domain)
//
//    slash_validator(state, proposer_slashing.proposer_index)
func ProcessProposerSlashings(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	var err error
	for idx, slashing := range body.ProposerSlashings {
		if slashing == nil {
			return nil, errors.New("nil proposer slashings in block body")
		}
		if err = VerifyProposerSlashing(beaconState, slashing); err != nil {
			return nil, errors.Wrapf(err, "could not verify proposer slashing %d", idx)
		}
		beaconState, err = v.SlashValidator(
			beaconState, slashing.Header_1.Header.ProposerIndex,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "could not slash proposer index %d", slashing.Header_1.Header.ProposerIndex)
		}
	}
	return beaconState, nil
}

// VerifyProposerSlashing verifies that the data provided from slashing is valid.
func VerifyProposerSlashing(
	beaconState *stateTrie.BeaconState,
	slashing *ethpb.ProposerSlashing,
) error {
	if slashing.Header_1 == nil || slashing.Header_1.Header == nil || slashing.Header_2 == nil || slashing.Header_2.Header == nil {
		return errors.New("nil header cannot be verified")
	}
	if slashing.Header_1.Header.Slot != slashing.Header_2.Header.Slot {
		return fmt.Errorf("mismatched header slots, received %d == %d", slashing.Header_1.Header.Slot, slashing.Header_2.Header.Slot)
	}
	if slashing.Header_1.Header.ProposerIndex != slashing.Header_2.Header.ProposerIndex {
		return fmt.Errorf("mismatched indices, received %d == %d", slashing.Header_1.Header.ProposerIndex, slashing.Header_2.Header.ProposerIndex)
	}
	if proto.Equal(slashing.Header_1, slashing.Header_2) {
		return errors.New("expected slashing headers to differ")
	}
	proposer, err := beaconState.ValidatorAtIndexReadOnly(slashing.Header_1.Header.ProposerIndex)
	if err != nil {
		return err
	}
	if !helpers.IsSlashableValidatorUsingTrie(proposer, helpers.SlotToEpoch(beaconState.Slot())) {
		return fmt.Errorf("validator with key %#x is not slashable", proposer.PublicKey())
	}
	// Using headerEpoch1 here because both of the headers should have the same epoch.
	domain, err := helpers.Domain(beaconState.Fork(), helpers.SlotToEpoch(slashing.Header_1.Header.Slot), params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		return err
	}
	headers := []*ethpb.SignedBeaconBlockHeader{slashing.Header_1, slashing.Header_2}
	for _, header := range headers {
		proposerPubKey := proposer.PublicKey()
		if err := helpers.VerifySigningRoot(header.Header, proposerPubKey[:], header.Signature, domain); err != nil {
			return errors.Wrap(err, "could not verify beacon block header")
		}
	}
	return nil
}
