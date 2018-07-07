package types

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// Shard defines a way for services attached to a sharding-enabled node to
// instantiate shards with a given ID and backend. This struct serves as
// an abstraction that contains useful methods to fetch collations corresponding to
// a shard from the DB, methods to check for data availability, and more.
type Shard struct {
	shardDB ethdb.Database
	shardID *big.Int
}

// NewShard creates an instance of a Shard struct given a shardID.
func NewShard(shardID *big.Int, shardDB ethdb.Database) *Shard {
	return &Shard{
		shardID: shardID,
		shardDB: shardDB,
	}
}

// ShardID gets the shard's unique identifier.
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

// HeaderByHash looks up a collation header from the shardDB using the header's hash.
func (s *Shard) HeaderByHash(hash *common.Hash) (*CollationHeader, error) {
	encoded, err := s.shardDB.Get(hash.Bytes())
	if err != nil {
		return nil, fmt.Errorf("get failed: %v", err)
	}
	if len(encoded) == 0 {
		return nil, fmt.Errorf("no value set for header hash: %vs", hash.Hex())
	}

	var header CollationHeader

	stream := rlp.NewStream(bytes.NewReader(encoded), uint64(len(encoded)))
	if err := header.DecodeRLP(stream); err != nil {
		return nil, fmt.Errorf("could not decode RLP header: %v", err)
	}

	return &header, nil
}

// CollationByHeaderHash fetches a collation by its header's hash from the DB.
func (s *Shard) CollationByHeaderHash(headerHash *common.Hash) (*Collation, error) {
	header, err := s.HeaderByHash(headerHash)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch header by hash: %v", err)
	}

	body, err := s.BodyByChunkRoot(header.ChunkRoot())
	if err != nil {
		return nil, fmt.Errorf("cannot fetch body by chunk root: %v", err)
	}

	txs, err := DeserializeBlobToTx(body)
	if err != nil {
		return nil, fmt.Errorf("cannot deserialize body: %v", err)
	}
	return NewCollation(header, body, *txs), nil
}

// ChunkRootfromHeaderHash gets the chunk root of a collation body from the hash of its header.
func (s *Shard) ChunkRootfromHeaderHash(headerHash *common.Hash) (*common.Hash, error) {
	collation, err := s.CollationByHeaderHash(headerHash)
	if err != nil {
		return nil, err
	}
	return collation.Header().ChunkRoot(), nil
}

// CanonicalHeaderHash gets a collation header hash that has been set as
// canonical for shardID/period pair.
func (s *Shard) CanonicalHeaderHash(shardID *big.Int, period *big.Int) (*common.Hash, error) {
	key := canonicalCollationLookupKey(shardID, period)

	// fetches the RLP encoded collation header corresponding to the key.
	encoded, err := s.shardDB.Get(key.Bytes())
	if err != nil {
		return nil, err
	}
	if len(encoded) == 0 {
		return nil, fmt.Errorf("no canonical collation header set for period=%v, shardID=%v pair: %v", shardID, period, err)
	}

	// RLP decodes the header, computes its hash.
	var header CollationHeader

	stream := rlp.NewStream(bytes.NewReader(encoded), uint64(len(encoded)))
	if err := header.DecodeRLP(stream); err != nil {
		return nil, fmt.Errorf("could not decode RLP header: %v", err)
	}

	collationHash := header.Hash()
	return &collationHash, nil
}

// CanonicalCollation fetches the collation set as canonical in the shardDB.
func (s *Shard) CanonicalCollation(shardID *big.Int, period *big.Int) (*Collation, error) {
	h, err := s.CanonicalHeaderHash(shardID, period)
	if err != nil {
		return nil, fmt.Errorf("error while getting canonical header hash: %v", err)
	}

	return s.CollationByHeaderHash(h)
}

// BodyByChunkRoot fetches a collation body given its chunk root.
func (s *Shard) BodyByChunkRoot(chunkRoot *common.Hash) ([]byte, error) {
	body, err := s.shardDB.Get(chunkRoot.Bytes())
	if err != nil {
		return []byte{}, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("no corresponding body with chunk root found: %s", chunkRoot)
	}
	return body, nil
}

// CheckAvailability is used by notaries to confirm a header's data availability.
func (s *Shard) CheckAvailability(header *CollationHeader) (bool, error) {
	key := dataAvailabilityLookupKey(header.ChunkRoot())
	availability, err := s.shardDB.Get(key.Bytes())
	if err != nil {
		return false, err
	}
	if len(availability) == 0 {
		return false, fmt.Errorf("availability not set for header")
	}
	// availability is a byte array of length 1.
	return availability[0] != 0, nil
}

// SetAvailability saves the availability of the chunk root in the shardDB.
func (s *Shard) SetAvailability(chunkRoot *common.Hash, availability bool) error {
	key := dataAvailabilityLookupKey(chunkRoot)
	var encoded []byte
	if availability {
		encoded = []byte{1}
	} else {
		encoded = []byte{0}
	}
	return s.shardDB.Put(key.Bytes(), encoded)
}

// SaveHeader adds the collation header to shardDB.
func (s *Shard) SaveHeader(header *CollationHeader) error {
	// checks if the header has a chunk root set before saving.
	if header.ChunkRoot() == nil {
		return fmt.Errorf("header needs to have a chunk root set before saving")
	}

	encoded, err := header.EncodeRLP()
	if err != nil {
		return fmt.Errorf("cannot encode header: %v", err)
	}

	// uses the hash of the header as the key.
	return s.shardDB.Put(header.Hash().Bytes(), encoded)
}

// SaveBody adds the collation body to the shardDB and sets availability.
func (s *Shard) SaveBody(body []byte) error {
	if len(body) == 0 {
		return errors.New("body is empty")
	}
	chunks := Chunks(body)                   // wrapper allowing us to merklizing the chunks.
	chunkRoot := coreTypes.DeriveSha(chunks) // merklize the serialized blobs.
	s.SetAvailability(&chunkRoot, true)
	return s.shardDB.Put(chunkRoot.Bytes(), body)
}

// SaveCollation adds the collation's header and body to shardDB.
func (s *Shard) SaveCollation(collation *Collation) error {
	if err := s.ValidateShardID(collation.Header()); err != nil {
		return err
	}
	if err := s.SaveHeader(collation.Header()); err != nil {
		return err
	}
	return s.SaveBody(collation.Body())
}

// SetCanonical sets the collation header as canonical in the shardDB. This is called
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

	// checks if the header has a corresponding body in the DB.
	body, err := s.BodyByChunkRoot(dbHeader.ChunkRoot())
	if err != nil {
		return fmt.Errorf("no corresponding collation body saved in shardDB: %v", body)
	}

	key := canonicalCollationLookupKey(dbHeader.ShardID(), dbHeader.Period())
	encoded, err := dbHeader.EncodeRLP()
	if err != nil {
		return fmt.Errorf("cannot encode header: %v", err)
	}
	// sets the key to be the canonical collation lookup key and val as RLP encoded
	// collation header.
	return s.shardDB.Put(key.Bytes(), encoded)
}

// dataAvailabilityLookupKey formats a string that will become a lookup
// key in the shardDB.
func dataAvailabilityLookupKey(chunkRoot *common.Hash) common.Hash {
	key := fmt.Sprintf("availability-lookup:%s", chunkRoot.Hex())
	return common.BytesToHash([]byte(key))
}

// canonicalCollationLookupKey formats a string that will become a lookup key
// in the shardDB that takes into account the shardID and the period
// of the shard for ease of use.
func canonicalCollationLookupKey(shardID *big.Int, period *big.Int) common.Hash {
	str := "canonical-collation-lookup:shardID=%s,period=%s"
	key := fmt.Sprintf(str, shardID, period)
	return common.BytesToHash([]byte(key))
}
