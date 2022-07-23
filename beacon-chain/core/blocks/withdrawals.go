package blocks

import (
	"bytes"
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash/htr"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
)

//
//def process_bls_to_execution_change(state: BeaconState,e: SignedBLSToExecutionChange) -> None:
//lidators)
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
func ProcessBLSToExecutionChange(ctx context.Context, st state.BeaconState, signed *ethpb.SignedBLSToExecutionChange) (state.BeaconState, error) {
	if signed == nil {
		return st, errNilSignedWithdrawalMessage
	}
	message := signed.Message
	if message == nil {
		return st, errNilSignedWithdrawalMessage
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
	newCredentials := make([]byte, 12)
	newCredentials[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
	val.WithdrawalCredentials = append(newCredentials, message.ToExecutionAddress...)
	return st, nil
}
