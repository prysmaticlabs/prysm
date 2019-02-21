package keystore

import (
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DepositInput for a given key. This input data can be used to when making a
// validator deposit. The input data includes a proof of possession field
// signed by the deposit key.
func DepositInput(depositKey *Key, withdrawalKey *Key) *pb.DepositInput {
	di := &pb.DepositInput{
		Pubkey:                      depositKey.PublicKey.Marshal(),
		WithdrawalCredentialsHash32: withdrawalCredentialsHash(withdrawalKey),
	}

	// #nosec G104 -- This can only throw if di is nil so lets ignore the error.
	b, _ := proto.Marshal(di)
	di.ProofOfPossession = depositKey.SecretKey.Sign(b, params.BeaconConfig().DomainDeposit).Marshal()

	return di
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
	return append([]byte{params.BeaconConfig().BLSWithdrawalPrefixByte}, h...)[:32]
}
