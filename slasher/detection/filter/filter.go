package filter

import (
	"encoding/binary"
	"hash/fnv"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/spaolacci/murmur3"
)

// BloomFilter is an encoded set of a []byte key.
type BloomFilter []byte

// NewBloomFilter returns a new bloom filter that encodes the given key with 16 bits allotted for it.
// This bloom filter has a set length of 16 bits, and uses 5 hash functions in order to provide less than
// 0.1% chance of false positives.
// The following 5 hash functions used are highway, murmur3, fnv, sha256, and keccak256. These are reliably quick
// hashes that used together can provide strong collision resistance.
func NewBloomFilter(key []byte) (BloomFilter, error) {
	nBits := 16
	filter := make([]byte, 2)
	for i := 0; i < len(filter); i++ {
		filter[i] = 0
	}

	// Getting the highway hash and setting its bit in the filter.
	hash1 := hashutil.FastSum64(key)
	bitPos := hash1 % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	// Getting the murmur3 hash and setting its bit in the filter.
	hash2 := murmur3.Sum64(key)
	bitPos = hash2 % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	// Getting the fnv hash and setting its bit in the filter.
	hash3 := fnv.New64()
	if _, err := hash3.Write(key); err != nil {
		return nil, err
	}
	bitPos = hash3.Sum64() % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	// Getting the sha256 hash and setting its bit in the filter.
	hash4 := hashutil.Hash(key)
	hash64 := binary.LittleEndian.Uint64(hash4[:])
	bitPos = hash64 % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	// Getting the keccak256 hash and setting its bit in the filter.
	hash5 := hashutil.HashKeccak256(key)
	hash64 = binary.LittleEndian.Uint64(hash5[:])
	bitPos = hash64 % uint64(nBits)
	filter[bitPos/8] |= 1 << (bitPos % 8)

	return filter, nil
}

// Contains returns whether the bloom filter contains given key. False positives
// are possible, where it could return true for a key not in the original set.
// This function generates the proper hashes for the key and checks if the
// corresponding bit is marked in the bloom filter.
func (f BloomFilter) Contains(key []byte) (bool, error) {
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

func (f BloomFilter) bitAt(bitPos uint64) bool {
	i := uint8(1 << (bitPos % 8))
	return f[bitPos/8]&i == i
}
