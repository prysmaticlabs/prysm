package sharding

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// Shard base struct.
type Shard struct {
	shardDB *shardBackend
	shardID *big.Int
}

// ValidateShardID checks if header belongs to shard.
func (s *Shard) ValidateShardID(h *CollationHeader) error {
	if s.shardID.Cmp(h.shardID) != 0 {
		return fmt.Errorf("Error: Collation Does Not Belong to Shard %d but Instead Has ShardID %d", h.shardID, s.shardID)
	}
	return nil
}

// SetHeader adds the collation header to shardDB
func (s *Shard) SetHeader(h *CollationHeader) error {
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

// GetBodyByChunkRoot fetches a collation body
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

// SetUnavailable ensures to set a collation as unavailable in the shardDB.
func (s *Shard) SetUnavailable(header *CollationHeader) error {
	return nil
}
