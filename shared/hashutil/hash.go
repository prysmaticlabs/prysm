package hashutil

import (
	"errors"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/minio/highwayhash"
	"github.com/minio/sha256-simd"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"golang.org/x/crypto/sha3"
)

// ErrNilProto can occur when attempting to hash a protobuf message that is nil
// or has nil objects within lists.
var ErrNilProto = errors.New("cannot hash a nil protobuf message")

// Hash defines a function that returns the sha256 checksum of the data passed in.
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#appendix
func Hash(data []byte) [32]byte {
	var hash [32]byte

	h := sha256.New()
	// The hash interface never returns an error, for that reason
	// we are not handling the error below. For reference, it is
	// stated here https://golang.org/pkg/hash/#Hash

	// #nosec G104
	h.Write(data)
	h.Sum(hash[:0])

	return hash
}

// HashKeccak256 defines a function which returns the Keccak-256/SHA3
// hash of the data passed in.
func HashKeccak256(data []byte) [32]byte {
	var hash [32]byte

	h := sha3.NewLegacyKeccak256()

	// The hash interface never returns an error, for that reason
	// we are not handling the error below. For reference, it is
	// stated here https://golang.org/pkg/hash/#Hash

	// #nosec G104
	h.Write(data)
	h.Sum(hash[:0])

	return hash
}

// RepeatHash applies the sha256 hash function repeatedly
// numTimes on a [32]byte array.
func RepeatHash(data [32]byte, numTimes uint64) [32]byte {
	if numTimes == 0 {
		return data
	}
	return RepeatHash(Hash(data[:]), numTimes-1)
}

// HashProto hashes a protocol buffer message using sha256.
func HashProto(msg proto.Message) (result [32]byte, err error) {
	// Hashing a proto with nil pointers will cause a panic in the unsafe
	// proto.Marshal library.
	defer func() {
		if r := recover(); r != nil {
			err = ErrNilProto
		}
	}()

	if msg == nil || reflect.ValueOf(msg).IsNil() {
		return [32]byte{}, ErrNilProto
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return [32]byte{}, err
	}
	return Hash(data), nil
}

// Key used for FastSum64
var fastSumHashKey = bytesutil.ToBytes32([]byte("hash_fast_sum64_key"))

// FastSum64 returns a hash sum of the input data using highwayhash. This method is not secure, but
// may be used as a quick identifier for objects where collisions are acceptable.
func FastSum64(data []byte) uint64 {
	return highwayhash.Sum64(data, fastSumHashKey[:])
}
