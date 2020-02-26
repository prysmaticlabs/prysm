package filter

import (
	"encoding/binary"
	"hash/fnv"

	"github.com/minio/blake2b-simd"
	"github.com/minio/highwayhash"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/spaolacci/murmur3"
)

// Filter is an encoded set of a []byte key.
type Filter []byte

// NewFilter returns a new Bloom filter that encodes the given key with 16 bits allotted for it.
func NewFilter(key []byte) (Filter, error) {
	nBits := 16
	nBytes := (nBits + 7) / 8
	nBits = nBytes * 8
	filter := make([]byte, nBytes)
	for i := 0; i < len(filter); i++ {
		filter[i] = 0
	}

	hash1, err := highwayhash.New64(key)
	if err != nil {
		return nil, err
	}
	bitPos := hash1.Sum64() % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	hash2 := murmur3.New64()
	_, err = hash2.Write(key)
	if err != nil {
		return nil, err
	}
	bitPos = hash2.Sum64() % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	hash3 := fnv.New64()
	_, err = hash3.Write(key)
	if err != nil {
		return nil, err
	}
	bitPos = hash3.Sum64() % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	hash4 := hashutil.Hash(key)
	hash64 := binary.LittleEndian.Uint64(hash4[:])
	bitPos = hash64 % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	hash5 := blake2b.New256()
	_, err = hash3.Write(key)
	if err != nil {
		return nil, err
	}
	hash64 = binary.LittleEndian.Uint64(hash5.Sum([]byte{})[:])
	bitPos = hash64 % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	return filter, nil
}

// Contains returns whether the filter contains given key. False positives
// are possible, where it could return true for a key not in the original set.
func (f Filter) Contains(key []byte) (bool, error) {
	if len(f) < 2 {
		return false, nil
	}
	nBits := uint64(16)

	hash1, err := highwayhash.New64(key)
	if err != nil {
		return false, err
	}
	if !f.bitAt(hash1.Sum64() % nBits) {
		return false, nil
	}

	hash2 := murmur3.New64()
	_, err = hash2.Write(key)
	if err != nil {
		return false, err
	}
	if !f.bitAt(hash2.Sum64() % nBits) {
		return false, nil
	}

	hash3 := fnv.New64()
	_, err = hash3.Write(key)
	if err != nil {
		return false, err
	}
	if !f.bitAt(hash3.Sum64() % nBits) {
		return false, nil
	}

	hash4 := hashutil.Hash(key)
	hash64 := binary.LittleEndian.Uint64(hash4[:])
	bitPos4 := hash64 % nBits
	if !f.bitAt(bitPos4) {
		return false, nil
	}

	hash5 := blake2b.New256()
	_, err = hash3.Write(key)
	if err != nil {
		return false, err
	}
	hash64 = binary.LittleEndian.Uint64(hash5.Sum([]byte{})[:])
	if !f.bitAt(hash64 % nBits) {
		return false, nil
	}
	return true, nil
}

func (filter Filter) bitAt(bitPos uint64) bool {
	i := uint8(1 << (bitPos % 8))
	if filter[bitPos/8]&i == i {
		return false
	}
	return true
}
