package keystore

import (
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DepositInput for a given key. This input data can be used to when making a
// validator deposit. The input data includes a proof of possession field
// signed by the deposit key.
//
// Spec details about general deposit workflow:
//   To submit a deposit:
//
//   - Pack the validator's initialization parameters into deposit_input, a DepositInput SSZ object.
//   - Set deposit_input.proof_of_possession = EMPTY_SIGNATURE.
//   - Let proof_of_possession be the result of bls_sign of the hash_tree_root(deposit_input) with domain=DOMAIN_DEPOSIT.
//   - Set deposit_input.proof_of_possession = proof_of_possession.
//   - Let amount be the amount in Gwei to be deposited by the validator where MIN_DEPOSIT_AMOUNT <= amount <= MAX_DEPOSIT_AMOUNT.
//   - Send a transaction on the Ethereum 1.0 chain to DEPOSIT_CONTRACT_ADDRESS executing deposit along with serialize(deposit_input) as the singular bytes input along with a deposit amount in Gwei.
//
// See: https://github.com/ethereum/eth2.0-specs/blob/dev/specs/validator/0_beacon-chain-validator.md#submit-deposit
func DepositInput(depositKey *Key, withdrawalKey *Key) (*pb.DepositData, error) {
	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte
	copy(pubkey[:], depositKey.PublicKey.Marshal())
	copy(withdrawalCreds[:], withdrawalCredentialsHash(withdrawalKey))
	di := &pb.DepositData{
		Pubkey:                pubkey[:],
		WithdrawalCredentials: withdrawalCreds[:],
	}

	buf, err := ssz.Marshal(di)
	if err != nil {
		return nil, err
	}
	domain := bytesutil.FromBytes4(params.BeaconConfig().DomainDeposit)
	di.Signature = depositKey.SecretKey.Sign(buf, domain).Marshal()

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
	h := Keccak256(withdrawalKey.PublicKey.Marshal())
	return append([]byte{params.BeaconConfig().BLSWithdrawalPrefixByte}, h[0:]...)[:32]
}
