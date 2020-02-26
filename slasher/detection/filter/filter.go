package filter

import (
	"encoding/binary"
	"hash/fnv"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/spaolacci/murmur3"
)

// Filter is an encoded set of a []byte key.
type Filter []byte

// NewFilter returns a new Bloom filter that encodes the given key with 16 bits allotted for it.
func NewFilter(key []byte) (Filter, error) {
	nBits := 16
	filter := make([]byte, 2)
	for i := 0; i < len(filter); i++ {
		filter[i] = 0
	}

	hash1 := hashutil.FastSum64(key)
	bitPos := hash1 % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	hash2 := murmur3.Sum64(key)
	bitPos = hash2 % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	hash3 := fnv.New64()
	if _, err := hash3.Write(key); err != nil {
		return nil, err
	}
	bitPos = hash3.Sum64() % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	hash4 := hashutil.Hash(key)
	hash64 := binary.LittleEndian.Uint64(hash4[:])
	bitPos = hash64 % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	hash5 := hashutil.HashKeccak256(key)
	hash64 = binary.LittleEndian.Uint64(hash5[:])
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

	hash1 := hashutil.FastSum64(key)
	if !f.bitAt(hash1 % nBits) {
		return false, nil
	}

	hash2 := murmur3.Sum64(key)
	if !f.bitAt(hash2 % nBits) {
		return false, nil
	}

	hash3 := fnv.New64()
	if _, err := hash3.Write(key); err != nil {
		return false, err
	}
	if !f.bitAt(hash3.Sum64() % nBits) {
		return false, nil
	}

	hash4 := hashutil.Hash(key)
	hash64 := binary.LittleEndian.Uint64(hash4[:])
	if !f.bitAt(hash64 % nBits) {
		return false, nil
	}

	hash5 := hashutil.HashKeccak256(key)
	hash64 = binary.LittleEndian.Uint64(hash5[:])
	if !f.bitAt(hash64 % nBits) {
		return false, nil
	}

	return true, nil
}

func (f Filter) bitAt(bitPos uint64) bool {
	i := uint8(1 << (bitPos % 8))
	return f[bitPos/8]&i == i
}
