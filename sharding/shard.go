package sharding

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// Shard base struct.
type Shard struct {
	shardDB *shardKV
	shardID *big.Int
}

// MakeShard creates an instance of a Shard struct given a shardID.
func MakeShard(shardID *big.Int) *Shard {
	// Swappable - can be makeShardLevelDB, makeShardSparseTrie, etc.
	shardDB := makeShardKV()

	return &Shard{
		shardID: shardID,
		shardDB: shardDB,
	}
}

// ValidateShardID checks if header belongs to shard.
func (s *Shard) ValidateShardID(h *CollationHeader) error {
	if s.shardID.Cmp(h.shardID) != 0 {
		return fmt.Errorf("Error: Collation Does Not Belong to Shard %d but Instead Has ShardID %d", h.shardID, s.shardID)
	}
	return nil
}

// SaveHeader adds the collation header to shardDB.
func (s *Shard) SaveHeader(h *CollationHeader) error {
	if err := s.ValidateShardID(h); err != nil {
		return err
	}
	encoded, err := rlp.EncodeToBytes(h)
	if err != nil {
		return fmt.Errorf("Error: Cannot Encode Header")
	}
	s.shardDB.Put(h.Hash(), encoded)
	return nil
}

// SaveCollationBody adds the collation body to shardDB.
func (s *Shard) SaveCollationBody(body []byte) error {
	// TODO: dependent on blob serialization.
	// chunkRoot := getChunkRoot(body) using the blob algorithm utils.
	// s.shardDB.Put(chunkRoot, body)
	// s.SetAvailability(chunkRoot, true)
	return nil
}

// GetHeaderByHash of collation.
func (s *Shard) GetHeaderByHash(hash common.Hash) (*CollationHeader, error) {
	encoded, err := s.shardDB.Get(hash)
	if err != nil {
		return nil, fmt.Errorf("Error: Header Not Found")
	}
	var header CollationHeader
	if err := rlp.DecodeBytes(encoded, &header); err != nil {
		return nil, fmt.Errorf("Error: Problem Decoding Header: %v", err)
	}
	return &header, nil
}

// GetCollationByHash fetches full collation.
func (s *Shard) GetCollationByHash(hash common.Hash) (*Collation, error) {
	header, err := s.GetHeaderByHash(hash)
	if err != nil {
		return nil, err
	}
	body, err := s.GetBodyByChunkRoot(*header.chunkRoot)
	if err != nil {
		return nil, err
	}
	return &Collation{header: header, body: body}, nil
}

// GetCanonicalCollationHash gets a collation header hash that has been set as canonical for
// shardID/period pair
func (s *Shard) GetCanonicalCollationHash(shardID *big.Int, period *big.Int) (*common.Hash, error) {
	key := canonicalCollationLookupKey(shardID, period)
	collationHashBytes, err := s.shardDB.Get(common.BytesToHash(key))
	if err != nil {
		return nil, fmt.Errorf("Error: No Canonical Collation Set for Period/ShardID")
	}
	collationHash := common.BytesToHash(collationHashBytes)
	return &collationHash, nil
}

// GetCanonicalCollation fetches the collation set as canonical in the shardDB.
func (s *Shard) GetCanonicalCollation(shardID *big.Int, period *big.Int) (*Collation, error) {
	h, err := s.GetCanonicalCollationHash(shardID, period)
	if err != nil {
		return nil, fmt.Errorf("Error: No Hash Found")
	}
	collation, err := s.GetCollationByHash(*h)
	if err != nil {
		return nil, fmt.Errorf("Error: No Canonical Collation Found for Hash")
	}
	return collation, nil
}

// GetBodyByChunkRoot fetches a collation body.
func (s *Shard) GetBodyByChunkRoot(chunkRoot common.Hash) ([]byte, error) {
	body, err := s.shardDB.Get(chunkRoot)
	if err != nil {
		return nil, fmt.Errorf("Error: No Corresponding Body With Chunk Root Found")
	}
	return body, nil
}

// CheckAvailability is used by notaries to confirm a header's data availability.
func (s *Shard) CheckAvailability(header *CollationHeader) bool {
	return true
}

// SetAvailability saves the availability of the chunk root in the shardDB.
func (s *Shard) SetAvailability(chunkRoot *common.Hash) error {
	return nil
}

// dataAvailabilityLookupKey formats a string that will become a lookup
// key in the shardDB.
func dataAvailabilityLookupKey(chunkRoot *common.Hash) []byte {
	return []byte(fmt.Sprintf("availability-lookup:%s", chunkRoot.Str()))
}

// dataAvailabilityLookupKey formats a string that will become a lookup key
// in the shardDB that takes into account the shardID and the period
// of the shard.
func canonicalCollationLookupKey(shardID *big.Int, period *big.Int) []byte {
	str := "canonical-collation-lookup:shardID=%s,period=%s"
	key := fmt.Sprintf(str, shardID.String(), period.String())
	return []byte(key)
}
