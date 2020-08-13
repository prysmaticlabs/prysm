package blocks

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessVoluntaryExits is one of the operations performed
// on each processed beacon block to determine which validators
// should exit the state's validator registry.
//
// Spec pseudocode definition:
//   def process_voluntary_exit(state: BeaconState, exit: VoluntaryExit) -> None:
//    """
//    Process ``VoluntaryExit`` operation.
//    """
//    validator = state.validator_registry[exit.validator_index]
//    # Verify the validator is active
//    assert is_active_validator(validator, get_current_epoch(state))
//    # Verify the validator has not yet exited
//    assert validator.exit_epoch == FAR_FUTURE_EPOCH
//    # Exits must specify an epoch when they become valid; they are not valid before then
//    assert get_current_epoch(state) >= exit.epoch
//    # Verify the validator has been active long enough
//    assert get_current_epoch(state) >= validator.activation_epoch + PERSISTENT_COMMITTEE_PERIOD
//    # Verify signature
//    domain = get_domain(state, DOMAIN_VOLUNTARY_EXIT, exit.epoch)
//    assert bls_verify(validator.pubkey, signing_root(exit), exit.signature, domain)
//    # Initiate exit
//    initiate_validator_exit(state, exit.validator_index)
func ProcessVoluntaryExits(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	exits := body.VoluntaryExits
	for idx, exit := range exits {
		if exit == nil || exit.Exit == nil {
			return nil, errors.New("nil voluntary exit in block body")
		}
		val, err := beaconState.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, err
		}
		if err := VerifyExit(val, beaconState.Slot(), beaconState.Fork(), exit, beaconState.GenesisValidatorRoot()); err != nil {
			return nil, errors.Wrapf(err, "could not verify exit %d", idx)
		}
		beaconState, err = v.InitiateValidatorExit(beaconState, exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, err
		}
	}
	return beaconState, nil
}

// ProcessVoluntaryExitsNoVerify processes all the voluntary exits in
// a block body, without verifying their BLS signatures.
func ProcessVoluntaryExitsNoVerify(
	beaconState *stateTrie.BeaconState,
	body *ethpb.BeaconBlockBody,
) (*stateTrie.BeaconState, error) {
	exits := body.VoluntaryExits

	for idx, exit := range exits {
		if exit == nil || exit.Exit == nil {
			return nil, errors.New("nil exit")
		}
		val, err := beaconState.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, err
		}
		if err := verifyExitConditions(val, beaconState.Slot(), exit.Exit); err != nil {
			return nil, err
		}
		// Validate that fork and genesis root are valid.
		_, err = helpers.Domain(beaconState.Fork(), exit.Exit.Epoch, params.BeaconConfig().DomainVoluntaryExit, beaconState.GenesisValidatorRoot())
		if err != nil {
			return nil, err
		}
		beaconState, err = v.InitiateValidatorExit(beaconState, exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process voluntary exit at index %d", idx)
		}
	}
	return beaconState, nil
}

// VerifyExit implements the spec defined validation for voluntary exits.
//
// Spec pseudocode definition:
//   def process_voluntary_exit(state: BeaconState, exit: VoluntaryExit) -> None:
//    """
//    Process ``VoluntaryExit`` operation.
//    """
//    validator = state.validator_registry[exit.validator_index]
//    # Verify the validator is active
//    assert is_active_validator(validator, get_current_epoch(state))
//    # Verify the validator has not yet exited
//    assert validator.exit_epoch == FAR_FUTURE_EPOCH
//    # Exits must specify an epoch when they become valid; they are not valid before then
//    assert get_current_epoch(state) >= exit.epoch
//    # Verify the validator has been active long enough
//    assert get_current_epoch(state) >= validator.activation_epoch + PERSISTENT_COMMITTEE_PERIOD
//    # Verify signature
//    domain = get_domain(state, DOMAIN_VOLUNTARY_EXIT, exit.epoch)
//    assert bls_verify(validator.pubkey, signing_root(exit), exit.signature, domain)
func VerifyExit(validator *stateTrie.ReadOnlyValidator, currentSlot uint64, fork *pb.Fork, signed *ethpb.SignedVoluntaryExit, genesisRoot []byte) error {
	if signed == nil || signed.Exit == nil {
		return errors.New("nil exit")
	}

	exit := signed.Exit
	if err := verifyExitConditions(validator, currentSlot, exit); err != nil {
		return err
	}
	domain, err := helpers.Domain(fork, exit.Epoch, params.BeaconConfig().DomainVoluntaryExit, genesisRoot)
	if err != nil {
		return err
	}
	valPubKey := validator.PublicKey()
	if err := helpers.VerifySigningRoot(exit, valPubKey[:], signed.Signature, domain); err != nil {
		return helpers.ErrSigFailedToVerify
	}
	return nil
}

// verifyExitConditions implements the spec defined validation for voluntary exits(excluding signatures).
//
// Spec pseudocode definition:
//   def process_voluntary_exit(state: BeaconState, exit: VoluntaryExit) -> None:
//    """
//    Process ``VoluntaryExit`` operation.
//    """
//    validator = state.validator_registry[exit.validator_index]
//    # Verify the validator is active
//    assert is_active_validator(validator, get_current_epoch(state))
//    # Verify the validator has not yet exited
//    assert validator.exit_epoch == FAR_FUTURE_EPOCH
//    # Exits must specify an epoch when they become valid; they are not valid before then
//    assert get_current_epoch(state) >= exit.epoch
//    # Verify the validator has been active long enough
//    assert get_current_epoch(state) >= validator.activation_epoch + SHARD_COMMITTEE_PERIOD
func verifyExitConditions(validator *stateTrie.ReadOnlyValidator, currentSlot uint64, exit *ethpb.VoluntaryExit) error {
	currentEpoch := helpers.SlotToEpoch(currentSlot)
	// Verify the validator is active.
	if !helpers.IsActiveValidatorUsingTrie(validator, currentEpoch) {
		return errors.New("non-active validator cannot exit")
	}
	// Verify the validator has not yet exited.
	if validator.ExitEpoch() != params.BeaconConfig().FarFutureEpoch {
		return fmt.Errorf("validator has already exited at epoch: %v", validator.ExitEpoch())
	}
	// Exits must specify an epoch when they become valid; they are not valid before then.
	if currentEpoch < exit.Epoch {
		return fmt.Errorf("expected current epoch >= exit epoch, received %d < %d", currentEpoch, exit.Epoch)
	}
	// Verify the validator has been active long enough.
	if currentEpoch < validator.ActivationEpoch()+params.BeaconConfig().ShardCommitteePeriod {
		return fmt.Errorf(
			"validator has not been active long enough to exit, wanted epoch %d >= %d",
			currentEpoch,
			validator.ActivationEpoch()+params.BeaconConfig().ShardCommitteePeriod,
		)
	}
	return nil
}
