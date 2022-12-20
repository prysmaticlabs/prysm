package blocks

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

const executionToBLSPadding = 12

func ProcessBLSToExecutionChanges(
	st state.BeaconState,
	signed interfaces.SignedBeaconBlock) (state.BeaconState, error) {
	if signed.Version() < version.Capella {
		return st, nil
	}
	changes, err := signed.Block().Body().BLSToExecutionChanges()
	if err != nil {
		return nil, errors.Wrap(err, "could not get BLSToExecutionChanges")
	}
	// Return early if no changes
	if len(changes) == 0 {
		return st, nil
	}
	for _, change := range changes {
		st, err = processBLSToExecutionChange(st, change)
		if err != nil {
			return nil, errors.Wrap(err, "could not process BLSToExecutionChange")
		}
	}
	return st, nil
}

// processBLSToExecutionChange validates a SignedBLSToExecution message and
// changes the validator's withdrawal address accordingly.
//
// Spec pseudocode definition:
//
// def process_bls_to_execution_change(state: BeaconState, signed_address_change: SignedBLSToExecutionChange) -> None:
//
//	validator = state.validators[address_change.validator_index]
//
//	assert validator.withdrawal_credentials[:1] == BLS_WITHDRAWAL_PREFIX
//	assert validator.withdrawal_credentials[1:] == hash(address_change.from_bls_pubkey)[1:]
//
//	domain = get_domain(state, DOMAIN_BLS_TO_EXECUTION_CHANGE)
//	signing_root = compute_signing_root(address_change, domain)
//	assert bls.Verify(address_change.from_bls_pubkey, signing_root, signed_address_change.signature)
//
//	validator.withdrawal_credentials = (
//	    ETH1_ADDRESS_WITHDRAWAL_PREFIX
//	    + b'\x00' * 11
//	    + address_change.to_execution_address
//	)
func processBLSToExecutionChange(st state.BeaconState, signed *ethpb.SignedBLSToExecutionChange) (state.BeaconState, error) {
	// Checks that the message passes the validation conditions.
	val, err := ValidateBLSToExecutionChange(st, signed)
	if err != nil {
		return nil, err
	}

	message := signed.Message
	newCredentials := make([]byte, executionToBLSPadding)
	newCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
	val.WithdrawalCredentials = append(newCredentials, message.ToExecutionAddress...)
	err = st.UpdateValidatorAtIndex(message.ValidatorIndex, val)
	return st, err
}

// ValidateBLSToExecutionChange validates the execution change message against the state and returns the
// validator referenced by the message.
func ValidateBLSToExecutionChange(st state.ReadOnlyBeaconState, signed *ethpb.SignedBLSToExecutionChange) (*ethpb.Validator, error) {
	if signed == nil {
		return nil, errNilSignedWithdrawalMessage
	}
	message := signed.Message
	if message == nil {
		return nil, errNilWithdrawalMessage
	}

	val, err := st.ValidatorAtIndex(message.ValidatorIndex)
	if err != nil {
		return nil, err
	}
	cred := val.WithdrawalCredentials
	if cred[0] != params.BeaconConfig().BLSWithdrawalPrefixByte {
		return nil, errInvalidBLSPrefix
	}

	// hash the public key and verify it matches the withdrawal credentials
	fromPubkey := message.FromBlsPubkey
	hashFn := ssz.NewHasherFunc(hash.CustomSHA256Hasher())
	digest := hashFn.Hash(fromPubkey)
	if !bytes.Equal(digest[1:], cred[1:]) {
		return nil, errInvalidWithdrawalCredentials
	}
	return val, nil
}

func ProcessWithdrawals(st state.BeaconState, withdrawals []*enginev1.Withdrawal) (state.BeaconState, error) {
	expected, err := st.ExpectedWithdrawals()
	if err != nil {
		return nil, errors.Wrap(err, "could not get expected withdrawals")
	}
	if len(expected) != len(withdrawals) {
		return nil, errInvalidWithdrawalNumber
	}
	for i, withdrawal := range withdrawals {
		if withdrawal.Index != expected[i].Index {
			return nil, errInvalidWithdrawalIndex
		}
		if withdrawal.ValidatorIndex != expected[i].ValidatorIndex {
			return nil, errInvalidValidatorIndex
		}
		if !bytes.Equal(withdrawal.Address, expected[i].Address) {
			return nil, errInvalidExecutionAddress
		}
		if withdrawal.Amount != expected[i].Amount {
			return nil, errInvalidWithdrawalAmount
		}
		err := helpers.DecreaseBalance(st, withdrawal.ValidatorIndex, withdrawal.Amount)
		if err != nil {
			return nil, errors.Wrap(err, "could not decrease balance")
		}
	}
	if len(withdrawals) > 0 {
		if err := st.SetNextWithdrawalIndex(withdrawals[len(withdrawals)-1].Index + 1); err != nil {
			return nil, errors.Wrap(err, "could not set next withdrawal index")
		}
	}
	var nextValidatorIndex types.ValidatorIndex
	if uint64(len(withdrawals)) < params.BeaconConfig().MaxWithdrawalsPerPayload {
		nextValidatorIndex, err = st.NextWithdrawalValidatorIndex()
		if err != nil {
			return nil, errors.Wrap(err, "could not get next withdrawal validator index")
		}
		nextValidatorIndex += types.ValidatorIndex(params.BeaconConfig().MaxValidatorsPerWithdrawalsSweep)
		nextValidatorIndex = nextValidatorIndex % types.ValidatorIndex(st.NumValidators())
	} else {
		nextValidatorIndex = withdrawals[len(withdrawals)-1].ValidatorIndex + 1
		if nextValidatorIndex == types.ValidatorIndex(st.NumValidators()) {
			nextValidatorIndex = 0
		}
	}
	if err := st.SetNextWithdrawalValidatorIndex(nextValidatorIndex); err != nil {
		return nil, errors.Wrap(err, "could not set next withdrawal validator index")
	}
	return st, nil
}

func BLSChangesSignatureBatch(
	st state.ReadOnlyBeaconState,
	changes []*ethpb.SignedBLSToExecutionChange,
) (*bls.SignatureBatch, error) {
	// Return early if no changes
	if len(changes) == 0 {
		return bls.NewSet(), nil
	}
	batch := &bls.SignatureBatch{
		Signatures:   make([][]byte, len(changes)),
		PublicKeys:   make([]bls.PublicKey, len(changes)),
		Messages:     make([][32]byte, len(changes)),
		Descriptions: make([]string, len(changes)),
	}
	epoch := slots.ToEpoch(st.Slot())
	var fork *ethpb.Fork
	if st.Version() < version.Capella {
		fork = &ethpb.Fork{
			PreviousVersion: params.BeaconConfig().BellatrixForkVersion,
			CurrentVersion:  params.BeaconConfig().CapellaForkVersion,
			Epoch:           params.BeaconConfig().CapellaForkEpoch,
		}
	} else {
		fork = st.Fork()
	}
	domain, err := signing.Domain(fork, epoch, params.BeaconConfig().DomainBLSToExecutionChange, st.GenesisValidatorsRoot())
	if err != nil {
		return nil, err
	}
	for i, change := range changes {
		batch.Signatures[i] = change.Signature
		publicKey, err := bls.PublicKeyFromBytes(change.Message.FromBlsPubkey)
		if err != nil {
			return nil, errors.Wrap(err, "could not convert bytes to public key")
		}
		batch.PublicKeys[i] = publicKey
		htr, err := signing.SigningData(change.Message.HashTreeRoot, domain)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute BLSToExecutionChange signing data")
		}
		batch.Messages[i] = htr
		batch.Descriptions[i] = signing.BlsChangeSignature
	}
	return batch, nil
}
