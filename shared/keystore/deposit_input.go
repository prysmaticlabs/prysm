package keystore

import (
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DepositInput for a given key. This input data can be used to when making a
// validator deposit. The input data includes a proof of possession field
// signed by the deposit key.
//
// Spec details about general deposit workflow:
//   To submit a deposit:
//
//   - Pack the validator's initialization parameters into deposit_data, a Deposit_Data SSZ object.
//   - Let amount be the amount in Gwei to be deposited by the validator where MIN_DEPOSIT_AMOUNT <= amount <= MAX_EFFECTIVE_BALANCE.
//   - Set deposit_data.amount = amount.
//   - Let signature be the result of bls_sign of the signing_root(deposit_data) with domain=compute_domain(DOMAIN_DEPOSIT). (Deposits are valid regardless of fork version, compute_domain will default to zeroes there).
//   - Send a transaction on the Ethereum 1.0 chain to DEPOSIT_CONTRACT_ADDRESS executing def deposit(pubkey: bytes[48], withdrawal_credentials: bytes[32], signature: bytes[96]) along with a deposit of amount Gwei.
//
// See: https://github.com/ethereum/eth2.0-specs/blob/master/specs/validator/0_beacon-chain-validator.md#submit-deposit
func DepositInput(depositKey *Key, withdrawalKey *Key, amountInGwei uint64) (*ethpb.Deposit_Data, error) {
	di := &ethpb.Deposit_Data{
		PublicKey:             depositKey.PublicKey.Marshal(),
		WithdrawalCredentials: withdrawalCredentialsHash(withdrawalKey),
		Amount:                amountInGwei,
	}

	sr, err := ssz.SigningRoot(di)
	if err != nil {
		return nil, err
	}

	domain := bls.Domain(params.BeaconConfig().DomainDeposit, params.BeaconConfig().GenesisForkVersion)
	di.Signature = depositKey.SecretKey.Sign(sr[:], domain).Marshal()

	return di, nil
}

// withdrawalCredentialsHash forms a 32 byte hash of the withdrawal public
// address.
//
// The specification is as follows:
//   withdrawal_credentials[:1] == BLS_WITHDRAWAL_PREFIX_BYTE
//   withdrawal_credentials[1:] == hash(withdrawal_pubkey)[1:]
// where withdrawal_credentials is of type bytes32.
func withdrawalCredentialsHash(withdrawalKey *Key) []byte {
	h := hashutil.Hash(withdrawalKey.PublicKey.Marshal())
	return append([]byte{params.BeaconConfig().BLSWithdrawalPrefixByte}, h[1:]...)[:32]
}
