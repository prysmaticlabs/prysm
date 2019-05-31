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

	pubkeyRoot := Hash(append(dep.Pubkey, zeroBytes[:16]...))
	sigHash := Hash(dep.Signature[:64])
	sigZeroBytesHash := Hash(append(dep.Signature[64:96], zeroBytes[:]...))
	sigRoot := Hash(append(sigHash[:], sigZeroBytesHash[:]...))

	pubRootWCredsHash := Hash(append(pubkeyRoot[:], dep.WithdrawalCredentials...))
	amountSigHash := Hash(append(append(bytesutil.Bytes8(dep.Amount), zeroBytes[:24]...), sigRoot[:]...))

	return Hash(append(pubRootWCredsHash[:], amountSigHash[:]...)), nil
}
