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

// ShardID gets the shard's identifier.
func (s *Shard) ShardID() *big.Int {
	return s.shardID
}

// ValidateShardID checks if header belongs to shard.
func (s *Shard) ValidateShardID(h *CollationHeader) error {
	if s.shardID.Cmp(h.shardID) != 0 {
		return fmt.Errorf("Error: Collation Does Not Belong to Shard %d but Instead Has ShardID %d", h.shardID, s.shardID)
	}
	return nil
}

// GetHeaderByHash of collation.
func (s *Shard) GetHeaderByHash(hash *common.Hash) (*CollationHeader, error) {
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
func (s *Shard) GetCollationByHash(hash *common.Hash) (*Collation, error) {
	header, err := s.GetHeaderByHash(hash)
	if err != nil {
		return nil, err
	}
	body, err := s.GetBodyByChunkRoot(header.ChunkRoot())
	if err != nil {
		return nil, err
	}
	return &Collation{header: header, body: body}, nil
}

// GetCanonicalCollationHash gets a collation header hash that has been set as canonical for
// shardID/period pair
func (s *Shard) GetCanonicalCollationHash(shardID *big.Int, period *big.Int) (*common.Hash, error) {
	key := canonicalCollationLookupKey(shardID, period)
	hash := common.BytesToHash(key.Bytes())
	collationHashBytes, err := s.shardDB.Get(&hash)
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
	collation, err := s.GetCollationByHash(h)
	if err != nil {
		return nil, fmt.Errorf("Error: No Canonical Collation Found for Hash")
	}
	return collation, nil
}

// GetBodyByChunkRoot fetches a collation body.
func (s *Shard) GetBodyByChunkRoot(chunkRoot *common.Hash) ([]byte, error) {
	body, err := s.shardDB.Get(chunkRoot)
	if err != nil {
		return nil, fmt.Errorf("Error: No Corresponding Body With Chunk Root Found")
	}
	return body, nil
}

// CheckAvailability is used by notaries to confirm a header's data availability.
func (s *Shard) CheckAvailability(header *CollationHeader) (bool, error) {
	key := dataAvailabilityLookupKey(header.ChunkRoot())
	availabilityVal, err := s.shardDB.Get(&key)
	if err != nil {
		return false, fmt.Errorf("Error: Key Not Found")
	}
	var availability int
	if err := rlp.DecodeBytes(availabilityVal, &availability); err != nil {
		return false, fmt.Errorf("Error: Cannot RLP Decode Availability: %v", err)
	}
	if availability != 0 {
		return true, nil
	}
	return false, nil
}

// SetAvailability saves the availability of the chunk root in the shardDB.
func (s *Shard) SetAvailability(chunkRoot *common.Hash, availability bool) error {
	key := dataAvailabilityLookupKey(chunkRoot)
	if availability {
		enc, err := rlp.EncodeToBytes(true)
		if err != nil {
			return fmt.Errorf("Cannot RLP encode availability: %v", err)
		}
		s.shardDB.Put(&key, enc)
	} else {
		enc, err := rlp.EncodeToBytes(false)
		if err != nil {
			return fmt.Errorf("Cannot RLP encode availability: %v", err)
		}
		s.shardDB.Put(&key, enc)
	}
	return nil
}

// SaveHeader adds the collation header to shardDB.
func (s *Shard) SaveHeader(header *CollationHeader) error {
	encoded, err := rlp.EncodeToBytes(header)
	if err != nil {
		return fmt.Errorf("Error: Cannot Encode Header")
	}
	// Uses the hash of the header as the key.
	hash := header.Hash()
	s.shardDB.Put(&hash, encoded)
	return nil
}

// SaveBody adds the collation body to the shardDB and sets availability.
func (s *Shard) SaveBody(body []byte) error {
	// TODO: dependent on blob serialization.
	// chunkRoot := getChunkRoot(body) using the blob algorithm utils.
	// right now we will just take the raw keccak256 of the body until #92 is merged.
	chunkRoot := common.BytesToHash(body)
	s.shardDB.Put(&chunkRoot, body)
	s.SetAvailability(&chunkRoot, true)
	return nil
}

// SaveCollation adds the collation's header and body to shardDB.
func (s *Shard) SaveCollation(collation *Collation) error {
	if err := s.ValidateShardID(collation.Header()); err != nil {
		return err
	}
	s.SaveHeader(collation.Header())
	s.SaveBody(collation.Body())
	return nil
}

// SetCanonical sets the collation as canonical in the shardDB. This is called
// after the period is over and over 2/3 notaries voted on the header.
func (s *Shard) SetCanonical(header *CollationHeader) error {
	if err := s.ValidateShardID(header); err != nil {
		return err
	}
	// the header needs to have been stored in the DB previously, so we
	// fetch it from the shardDB.
	hash := header.Hash()
	dbHeader, err := s.GetHeaderByHash(&hash)
	if err != nil {
		return err
	}
	key := canonicalCollationLookupKey(dbHeader.ShardID(), dbHeader.Period())
	encoded, err := rlp.EncodeToBytes(dbHeader)
	if err != nil {
		return fmt.Errorf("Error: Cannot Encode Header")
	}
	s.shardDB.Put(&key, encoded)
	return nil
}

// dataAvailabilityLookupKey formats a string that will become a lookup
// key in the shardDB.
func dataAvailabilityLookupKey(chunkRoot *common.Hash) common.Hash {
	key := fmt.Sprintf("availability-lookup:%s", chunkRoot.Str())
	return common.BytesToHash([]byte(key))
}

// dataAvailabilityLookupKey formats a string that will become a lookup key
// in the shardDB that takes into account the shardID and the period
// of the shard for ease of use.
func canonicalCollationLookupKey(shardID *big.Int, period *big.Int) common.Hash {
	str := "canonical-collation-lookup:shardID=%s,period=%s"
	key := fmt.Sprintf(str, shardID.String(), period.String())
	return common.BytesToHash([]byte(key))
}
