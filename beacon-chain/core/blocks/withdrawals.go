package blocks

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash/htr"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

const executionToBLSPadding = 12

// ProcessBLSToExecutionChange validates a SignedBLSToExecution message and
// changes the validator's withdrawal address accordingly.
//
// Spec pseudocode definition:
//
//def process_bls_to_execution_change(state: BeaconState, signed_address_change: SignedBLSToExecutionChange) -> None:
//    validator = state.validators[address_change.validator_index]
//
//    assert validator.withdrawal_credentials[:1] == BLS_WITHDRAWAL_PREFIX
//    assert validator.withdrawal_credentials[1:] == hash(address_change.from_bls_pubkey)[1:]
//
//    domain = get_domain(state, DOMAIN_BLS_TO_EXECUTION_CHANGE)
//    signing_root = compute_signing_root(address_change, domain)
//    assert bls.Verify(address_change.from_bls_pubkey, signing_root, signed_address_change.signature)
//
//    validator.withdrawal_credentials = (
//        ETH1_ADDRESS_WITHDRAWAL_PREFIX
//        + b'\x00' * 11
//        + address_change.to_execution_address
//    )
//
func ProcessBLSToExecutionChange(st state.BeaconState, signed *ethpb.SignedBLSToExecutionChange) (state.BeaconState, error) {
	if signed == nil {
		return st, errNilSignedWithdrawalMessage
	}
	message := signed.Message
	if message == nil {
		return st, errNilWithdrawalMessage
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
	pubkeyChunks := [][32]byte{bytesutil.ToBytes32(fromPubkey[:32]), bytesutil.ToBytes32(fromPubkey[32:])}
	digest := make([][32]byte, 1)
	htr.VectorizedSha256(pubkeyChunks, digest)
	if !bytes.Equal(digest[0][1:], cred[1:]) {
		return nil, errInvalidWithdrawalCredentials
	}

	epoch := slots.ToEpoch(st.Slot())
	domain, err := signing.Domain(st.Fork(), epoch, params.BeaconConfig().DomainBLSToExecutionChange, st.GenesisValidatorsRoot())
	if err != nil {
		return nil, err
	}
	if err := signing.VerifySigningRoot(message, fromPubkey, signed.Signature, domain); err != nil {
		return nil, signing.ErrSigFailedToVerify
	}
	newCredentials := make([]byte, executionToBLSPadding)
	newCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
	val.WithdrawalCredentials = append(newCredentials, message.ToExecutionAddress...)
	err = st.UpdateValidatorAtIndex(message.ValidatorIndex, val)
	return st, err
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
		if withdrawal.WithdrawalIndex != expected[i].WithdrawalIndex {
			return nil, errInvalidWithdrawalIndex
		}
		if withdrawal.ValidatorIndex != expected[i].ValidatorIndex {
			return nil, errInvalidValidatorIndex
		}
		if !bytes.Equal(withdrawal.ExecutionAddress, expected[i].ExecutionAddress) {
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
		if err := st.SetNextWithdrawalIndex(withdrawals[len(withdrawals)-1].WithdrawalIndex + 1); err != nil {
			return nil, errors.Wrap(err, "could not set withdrawal index")
		}
		if err := st.SetLastWithdrawalValidatorIndex(withdrawals[len(withdrawals)-1].ValidatorIndex); err != nil {
			return nil, errors.Wrap(err, "could not set latest withdrawal validator index")
		}
	}
	return st, nil
}
