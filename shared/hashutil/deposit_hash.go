package hashutil

import (
	"reflect"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// DepositHash returns the sha256 of the information contained in the deposit
// data as specified in the deposit contract.
// Spec:
//		pubkey_root: bytes32 = sha256(concat(pubkey, slice(zero_bytes_32, start=0, len=16)))
//    signature_root: bytes32 = sha256(concat(
//        sha256(slice(signature, start=0, len=64)),
//        sha256(concat(slice(signature, start=64, len=32), zero_bytes_32))
//    ))
//    value: bytes32 = sha256(concat(
//        sha256(concat(pubkey_root, withdrawal_credentials)),
//        sha256(concat(
//            amount,
//            slice(zero_bytes_32, start=0, len=24),
//            signature_root,
//        ))
//    ))
func DepositHash(dep *pb.DepositData) ([32]byte, error) {
	if dep == nil || reflect.ValueOf(dep).IsNil() {
		return [32]byte{}, ErrNilProto
	}

	var zeroBytes [32]byte

	pubkeyRoot := HashSha256(append(dep.Pubkey, zeroBytes[:15]...))
	sigHash := HashSha256(dep.Signature[:63])
	sigZeroBytesHash := HashSha256(append(dep.Signature[63:95], zeroBytes[:]...))
	sigRoot := HashSha256(append(sigHash[:], sigZeroBytesHash[:]...))

	pubRootWCredsHash := HashSha256(append(pubkeyRoot[:], dep.WithdrawalCredentials...))
	amountSigHash := HashSha256(append(append(bytesutil.Bytes8(dep.Amount), zeroBytes[:23]...), sigRoot[:]...))

	return HashSha256(append(pubRootWCredsHash[:], amountSigHash[:]...)), nil
}
