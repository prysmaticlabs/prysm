package interop

import (
	"encoding/binary"
	"math/big"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

const (
	blsWithdrawalPrefixByte = byte(0)
)

// DeterministicallyGenerateKeys creates BLS private keys using a fixed curve order according to
// the algorithm specified in the Eth2.0-Specs interop mock start section found here:
// https://github.com/ethereum/eth2.0-pm/blob/a085c9870f3956d6228ed2a40cd37f0c6580ecd7/interop/mocked_start/README.md
func DeterministicallyGenerateKeys(startIndex, numKeys uint64) ([]*bls.SecretKey, []*bls.PublicKey, error) {
	privKeys := make([]*bls.SecretKey, numKeys)
	pubKeys := make([]*bls.PublicKey, numKeys)
	for i := startIndex; i < startIndex+numKeys; i++ {
		enc := make([]byte, 32)
		binary.LittleEndian.PutUint32(enc, uint32(i))
		hash := hashutil.Hash(enc)
		// Reverse byte order to big endian for use with big ints.
		b := reverseByteOrder(hash[:])
		num := new(big.Int)
		num = num.SetBytes(b)
		order := new(big.Int)
		var ok bool
		order, ok = order.SetString(bls.CurveOrder, 10)
		if !ok {
			return nil, nil, errors.New("could not set bls curve order as big int")
		}
		num = num.Mod(num, order)
		numBytes := num.Bytes()
		// pad key at the start with zero bytes to make it into a 32 byte key
		if len(numBytes) < 32 {
			emptyBytes := make([]byte, 32-len(numBytes))
			numBytes = append(emptyBytes, numBytes...)
		}
		priv, err := bls.SecretKeyFromBytes(numBytes)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not create bls secret key at index %d from raw bytes", i)
		}
		privKeys[i-startIndex] = priv
		pubKeys[i-startIndex] = priv.PublicKey()
	}
	return privKeys, pubKeys, nil
}

// Switch the endianness of a byte slice by reversing its order.
func reverseByteOrder(input []byte) []byte {
	b := input
	for i := 0; i < len(b)/2; i++ {
		b[i], b[len(b)-i-1] = b[len(b)-i-1], b[i]
	}
	return b
}
