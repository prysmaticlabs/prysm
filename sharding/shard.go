package sharding

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type shardBackend interface {
	Get(k common.Hash) ([]byte, error)
	Has(k common.Hash) bool
	Put(k common.Hash, val []byte)
	Delete(k common.Hash)
}

// Shard base struct.
type Shard struct {
	shardDB shardBackend
	shardID *big.Int
}

// MakeShard creates an instance of a Shard struct given a shardID.
func MakeShard(shardID *big.Int, shardDB shardBackend) *Shard {
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
	if s.ShardID().Cmp(h.ShardID()) != 0 {
		return fmt.Errorf("collation does not belong to shard %d but instead has shardID %d", h.ShardID().Int64(), s.ShardID().Int64())
	}
	return nil
}

// HeaderByHash of collation.
func (s *Shard) HeaderByHash(hash *common.Hash) (*CollationHeader, error) {
	encoded, err := s.shardDB.Get(*hash)
	if err != nil {
		return nil, fmt.Errorf("header not found: %v", err)
	}

	var header CollationHeader

	stream := rlp.NewStream(bytes.NewReader(encoded), uint64(len(encoded)))
	if err := header.DecodeRLP(stream); err != nil {
		return nil, fmt.Errorf("could not decode RLP header: %v", err)
	}

	return &header, nil
}

// CollationByHash fetches full collation.
func (s *Shard) CollationByHash(headerHash *common.Hash) (*Collation, error) {
	header, err := s.HeaderByHash(headerHash)
	if err != nil {
		return nil, err
	}

	body, err := s.BodyByChunkRoot(header.ChunkRoot())
	if err != nil {
		return nil, err
	}
	return &Collation{header: header, body: body}, nil
}

// CanonicalCollationHash gets a collation header hash that has been set as canonical for
// shardID/period pair
func (s *Shard) CanonicalCollationHash(shardID *big.Int, period *big.Int) (*common.Hash, error) {
	key := canonicalCollationLookupKey(shardID, period)
	hash := common.BytesToHash(key.Bytes())
	collationHashBytes, err := s.shardDB.Get(hash)
	if err != nil {
		return nil, fmt.Errorf("no canonical collation set for period, shardID pair: %v", err)
	}
	collationHash := common.BytesToHash(collationHashBytes)
	return &collationHash, nil
}

// CanonicalCollation fetches the collation set as canonical in the shardDB.
func (s *Shard) CanonicalCollation(shardID *big.Int, period *big.Int) (*Collation, error) {
	h, err := s.CanonicalCollationHash(shardID, period)
	if err != nil {
		return nil, fmt.Errorf("hash not found: %v", err)
	}
	collation, err := s.CollationByHash(h)
	if err != nil {
		return nil, fmt.Errorf("no canonical collation found for hash: %v", err)
	}
	return collation, nil
}

// BodyByChunkRoot fetches a collation body.
func (s *Shard) BodyByChunkRoot(chunkRoot *common.Hash) ([]byte, error) {
	body, err := s.shardDB.Get(*chunkRoot)
	if err != nil {
		return nil, fmt.Errorf("no corresponding body with chunk root found: %v", err)
	}
	return body, nil
}

// CheckAvailability is used by notaries to confirm a header's data availability.
func (s *Shard) CheckAvailability(header *CollationHeader) (bool, error) {
	key := dataAvailabilityLookupKey(header.ChunkRoot())
	availabilityVal, err := s.shardDB.Get(key)
	if err != nil {
		return false, fmt.Errorf("key not found: %v", key)
	}
	var availability int
	if err := rlp.DecodeBytes(availabilityVal, &availability); err != nil {
		return false, fmt.Errorf("cannot RLP decode availability: %v", err)
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
			return fmt.Errorf("cannot RLP encode availability: %v", err)
		}
		s.shardDB.Put(key, enc)
	} else {
		enc, err := rlp.EncodeToBytes(false)
		if err != nil {
			return fmt.Errorf("cannot RLP encode availability: %v", err)
		}
		s.shardDB.Put(key, enc)
	}
	return nil
}

// SaveHeader adds the collation header to shardDB.
func (s *Shard) SaveHeader(header *CollationHeader) error {
	encoded, err := header.EncodeRLP()
	if err != nil {
		return fmt.Errorf("cannot encode header: %v", err)
	}

	// Uses the hash of the header as the key.
	hash := header.Hash()
	s.shardDB.Put(hash, encoded)
	return nil
}

// SaveBody adds the collation body to the shardDB and sets availability.
func (s *Shard) SaveBody(body []byte) error {
	// TODO: check if body is empty and throw error.
	// TODO: dependent on blob serialization.
	// chunkRoot := getChunkRoot(body) using the blob algorithm utils.
	// right now we will just take the raw keccak256 of the body until #92 is merged.
	chunkRoot := common.BytesToHash(body)
	s.shardDB.Put(chunkRoot, body)
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
	dbHeader, err := s.HeaderByHash(&hash)
	if err != nil {
		return err
	}
	key := canonicalCollationLookupKey(dbHeader.ShardID(), dbHeader.Period())
	encoded, err := dbHeader.EncodeRLP()
	if err != nil {
		return fmt.Errorf("cannot encode header: %v", err)
	}
	s.shardDB.Put(key, encoded)
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
