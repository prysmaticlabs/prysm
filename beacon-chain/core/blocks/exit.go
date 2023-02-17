package blocks

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	v "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// ValidatorAlreadyExitedMsg defines a message saying that a validator has already exited.
var ValidatorAlreadyExitedMsg = "has already submitted an exit, which will take place at epoch"

// ValidatorCannotExitYetMsg defines a message saying that a validator cannot exit
// because it has not been active long enough.
var ValidatorCannotExitYetMsg = "validator has not been active long enough to exit"

// ProcessVoluntaryExits is one of the operations performed
// on each processed beacon block to determine which validators
// should exit the state's validator registry.
//
// Spec pseudocode definition:
//
//	def process_voluntary_exit(state: BeaconState, signed_voluntary_exit: SignedVoluntaryExit) -> None:
//	 voluntary_exit = signed_voluntary_exit.message
//	 validator = state.validators[voluntary_exit.validator_index]
//	 # Verify the validator is active
//	 assert is_active_validator(validator, get_current_epoch(state))
//	 # Verify exit has not been initiated
//	 assert validator.exit_epoch == FAR_FUTURE_EPOCH
//	 # Exits must specify an epoch when they become valid; they are not valid before then
//	 assert get_current_epoch(state) >= voluntary_exit.epoch
//	 # Verify the validator has been active long enough
//	 assert get_current_epoch(state) >= validator.activation_epoch + SHARD_COMMITTEE_PERIOD
//	 # Verify signature
//	 domain = get_domain(state, DOMAIN_VOLUNTARY_EXIT, voluntary_exit.epoch)
//	 signing_root = compute_signing_root(voluntary_exit, domain)
//	 assert bls.Verify(validator.pubkey, signing_root, signed_voluntary_exit.signature)
//	 # Initiate exit
//	 initiate_validator_exit(state, voluntary_exit.validator_index)
func ProcessVoluntaryExits(
	ctx context.Context,
	beaconState state.BeaconState,
	exits []*ethpb.SignedVoluntaryExit,
) (state.BeaconState, error) {
	for idx, exit := range exits {
		if exit == nil || exit.Exit == nil {
			return nil, errors.New("nil voluntary exit in block body")
		}
		val, err := beaconState.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, err
		}
		if err := VerifyExitAndSignature(val, beaconState.Slot(), beaconState.Fork(), exit, beaconState.GenesisValidatorsRoot()); err != nil {
			return nil, errors.Wrapf(err, "could not verify exit %d", idx)
		}
		beaconState, err = v.InitiateValidatorExit(ctx, beaconState, exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, err
		}
	}
	return beaconState, nil
}

// VerifyExitAndSignature implements the spec defined validation for voluntary exits.
//
// Spec pseudocode definition:
//
//	def process_voluntary_exit(state: BeaconState, signed_voluntary_exit: SignedVoluntaryExit) -> None:
//	 voluntary_exit = signed_voluntary_exit.message
//	 validator = state.validators[voluntary_exit.validator_index]
//	 # Verify the validator is active
//	 assert is_active_validator(validator, get_current_epoch(state))
//	 # Verify exit has not been initiated
//	 assert validator.exit_epoch == FAR_FUTURE_EPOCH
//	 # Exits must specify an epoch when they become valid; they are not valid before then
//	 assert get_current_epoch(state) >= voluntary_exit.epoch
//	 # Verify the validator has been active long enough
//	 assert get_current_epoch(state) >= validator.activation_epoch + SHARD_COMMITTEE_PERIOD
//	 # Verify signature
//	 domain = get_domain(state, DOMAIN_VOLUNTARY_EXIT, voluntary_exit.epoch)
//	 signing_root = compute_signing_root(voluntary_exit, domain)
//	 assert bls.Verify(validator.pubkey, signing_root, signed_voluntary_exit.signature)
//	 # Initiate exit
//	 initiate_validator_exit(state, voluntary_exit.validator_index)
func VerifyExitAndSignature(
	validator state.ReadOnlyValidator,
	currentSlot primitives.Slot,
	fork *ethpb.Fork,
	signed *ethpb.SignedVoluntaryExit,
	genesisRoot []byte,
) error {
	if signed == nil || signed.Exit == nil {
		return errors.New("nil exit")
	}

	exit := signed.Exit
	if err := VerifyExitConditions(validator, currentSlot, exit); err != nil {
		return err
	}
	domain, err := signing.Domain(fork, exit.Epoch, params.BeaconConfig().DomainVoluntaryExit, genesisRoot)
	if err != nil {
		return err
	}
	valPubKey := validator.PublicKey()
	if err := signing.VerifySigningRoot(exit, valPubKey[:], signed.Signature, domain); err != nil {
		return signing.ErrSigFailedToVerify
	}
	return nil
}

// VerifyExitConditions implements the spec defined validation for voluntary exits (excluding signatures).
//
// Spec pseudocode definition:
//
//	def process_voluntary_exit(state: BeaconState, signed_voluntary_exit: SignedVoluntaryExit) -> None:
//	 voluntary_exit = signed_voluntary_exit.message
//	 validator = state.validators[voluntary_exit.validator_index]
//	 # Verify the validator is active
//	 assert is_active_validator(validator, get_current_epoch(state))
//	 # Verify exit has not been initiated
//	 assert validator.exit_epoch == FAR_FUTURE_EPOCH
//	 # Exits must specify an epoch when they become valid; they are not valid before then
//	 assert get_current_epoch(state) >= voluntary_exit.epoch
//	 # Verify the validator has been active long enough
//	 assert get_current_epoch(state) >= validator.activation_epoch + SHARD_COMMITTEE_PERIOD
//	 # Verify signature
//	 domain = get_domain(state, DOMAIN_VOLUNTARY_EXIT, voluntary_exit.epoch)
//	 signing_root = compute_signing_root(voluntary_exit, domain)
//	 assert bls.Verify(validator.pubkey, signing_root, signed_voluntary_exit.signature)
//	 # Initiate exit
//	 initiate_validator_exit(state, voluntary_exit.validator_index)
func VerifyExitConditions(validator state.ReadOnlyValidator, currentSlot primitives.Slot, exit *ethpb.VoluntaryExit) error {
	currentEpoch := slots.ToEpoch(currentSlot)
	// Verify the validator is active.
	if !helpers.IsActiveValidatorUsingTrie(validator, currentEpoch) {
		return errors.New("non-active validator cannot exit")
	}
	// Verify the validator has not yet submitted an exit.
	if validator.ExitEpoch() != params.BeaconConfig().FarFutureEpoch {
		return fmt.Errorf("validator with index %d %s: %v", exit.ValidatorIndex, ValidatorAlreadyExitedMsg, validator.ExitEpoch())
	}
	// Exits must specify an epoch when they become valid; they are not valid before then.
	if currentEpoch < exit.Epoch {
		return fmt.Errorf("expected current epoch >= exit epoch, received %d < %d", currentEpoch, exit.Epoch)
	}
	// Verify the validator has been active long enough.
	if currentEpoch < validator.ActivationEpoch()+params.BeaconConfig().ShardCommitteePeriod {
		return fmt.Errorf(
			"%s: %d of %d epochs. Validator will be eligible for exit at epoch %d",
			ValidatorCannotExitYetMsg,
			currentEpoch-validator.ActivationEpoch(),
			params.BeaconConfig().ShardCommitteePeriod,
			validator.ActivationEpoch()+params.BeaconConfig().ShardCommitteePeriod,
		)
	}
	return nil
}

// ExitSignatureBatch prepares a set of signatures, one for every passed in voluntary exit.
func ExitSignatureBatch(
	st state.ReadOnlyBeaconState,
	exits []*ethpb.SignedVoluntaryExit,
) (*bls.SignatureBatch, error) {
	// Return early if no exits
	if len(exits) == 0 {
		return bls.NewSet(), nil
	}
	batch := &bls.SignatureBatch{
		Signatures:   make([][]byte, len(exits)),
		PublicKeys:   make([]bls.PublicKey, len(exits)),
		Messages:     make([][32]byte, len(exits)),
		Descriptions: make([]string, len(exits)),
	}
	domain, err := signing.Domain(st.Fork(), slots.ToEpoch(st.Slot()), params.BeaconConfig().DomainVoluntaryExit, st.GenesisValidatorsRoot())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute signing domain")
	}
	for i, exit := range exits {
		batch.Signatures[i] = exit.Signature
		validator, err := st.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
		if err != nil {
			return nil, fmt.Errorf("could not obtain validator at index %d", exit.Exit.ValidatorIndex)
		}
		publicKeyBytes := validator.PublicKey()
		publicKey, err := bls.PublicKeyFromBytes(publicKeyBytes[:])
		if err != nil {
			return nil, errors.Wrap(err, "could not convert bytes to public key")
		}
		batch.PublicKeys[i] = publicKey
		htr, err := signing.SigningData(exit.Exit.HashTreeRoot, domain)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute exit's signing data")
		}
		batch.Messages[i] = htr
		batch.Descriptions[i] = signing.VoluntaryExitSignature
	}
	return batch, nil
}
