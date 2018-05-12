package sharding

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/sharding/database"
)

// Hash returns the hash of a collation's entire contents. Useful for comparison tests.
func (c *Collation) Hash() (hash common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, c)
	hw.Sum(hash[:0])
	return hash
}
func TestShard_ValidateShardID(t *testing.T) {
	emptyHash := common.StringToHash("")
	emptyAddr := common.StringToAddress("")
	header := NewCollationHeader(big.NewInt(1), &emptyHash, big.NewInt(1), &emptyAddr, []byte{})
	shardDB := database.MakeShardKV()
	shard := MakeShard(big.NewInt(3), shardDB)

	if err := shard.ValidateShardID(header); err == nil {
		t.Errorf("ShardID validation incorrect. Function should throw error when ShardID's do not match. want=%d. got=%d", header.ShardID().Int64(), shard.ShardID().Int64())
	}

	header2 := NewCollationHeader(big.NewInt(100), &emptyHash, big.NewInt(1), &emptyAddr, []byte{})
	shard2 := MakeShard(big.NewInt(100), shardDB)

	if err := shard2.ValidateShardID(header2); err != nil {
		t.Errorf("ShardID validation incorrect. Function should not throw error when ShardID's match. want=%d. got=%d", header2.ShardID().Int64(), shard2.ShardID().Int64())
	}
}

func TestShard_HeaderByHash(t *testing.T) {
	emptyHash := common.StringToHash("")
	emptyAddr := common.StringToAddress("")
	header := NewCollationHeader(big.NewInt(1), &emptyHash, big.NewInt(1), &emptyAddr, []byte{})
	shardDB := database.MakeShardKV()
	shard := MakeShard(big.NewInt(1), shardDB)

	if err := shard.SaveHeader(header); err != nil {
		t.Fatalf("cannot save collation header: %v", err)
	}
	hash := header.Hash()

	dbHeader, err := shard.HeaderByHash(&hash)
	if err != nil {
		t.Fatalf("could not fetch collation header by hash: %v", err)
	}
	// Compare the hashes.
	if header.Hash() != dbHeader.Hash() {
		t.Errorf("headers do not match. want=%v. got=%v", header, dbHeader)
	}
}

func TestShard_CollationByHash(t *testing.T) {
	emptyAddr := common.StringToAddress("")

	// Empty chunk root.
	header := NewCollationHeader(big.NewInt(1), nil, big.NewInt(1), &emptyAddr, []byte{})

	collation := &Collation{
		header: header,
		body:   []byte{1, 2, 3},
	}

	// We set the chunk root.
	collation.CalculateChunkRoot()

	shardDB := database.MakeShardKV()
	shard := MakeShard(big.NewInt(1), shardDB)

	if err := shard.SaveCollation(collation); err != nil {
		t.Fatalf("cannot save collation: %v", err)
	}
	hash := collation.Header().Hash()

	dbCollation, err := shard.CollationByHash(&hash)
	if err != nil {
		t.Fatalf("could not fetch collation by hash: %v", err)
	}

	// Compare the hashes.
	if collation.Hash() != dbCollation.Hash() {
		t.Errorf("collations do not match. want=%v. got=%v", collation, dbCollation)
	}
}

func TestShard_CanonicalHeaderHash(t *testing.T) {
	shardID := big.NewInt(1)
	period := big.NewInt(1)
	proposerAddress := common.StringToAddress("")
	proposerSignature := []byte{}
	emptyHash := common.StringToHash("")
	header := NewCollationHeader(shardID, &emptyHash, period, &proposerAddress, proposerSignature)

	shardDB := database.MakeShardKV()
	shard := MakeShard(shardID, shardDB)

	if err := shard.SaveHeader(header); err != nil {
		t.Fatalf("failed to save header to shardDB: %v", err)
	}

	if err := shard.SetCanonical(header); err != nil {
		t.Fatalf("failed to set header as canonical: %v", err)
	}

	headerHash := header.Hash()

	canonicalHeaderHash, err := shard.CanonicalHeaderHash(shardID, period)
	if err != nil {
		t.Fatalf("failed to get canonical header hash from shardDB: %v", err)
	}

	if canonicalHeaderHash.String() != headerHash.String() {
		t.Errorf("header hashes do not match. want=%v. got=%v", headerHash.String(), canonicalHeaderHash.String())
	}
}

func TestShard_CanonicalCollation(t *testing.T) {
	shardID := big.NewInt(1)
	period := big.NewInt(1)
	proposerAddress := common.StringToAddress("")
	proposerSignature := []byte{}
	emptyHash := common.StringToHash("")
	header := NewCollationHeader(shardID, &emptyHash, period, &proposerAddress, proposerSignature)

	shardDB := database.MakeShardKV()
	shard := MakeShard(shardID, shardDB)

	collation := &Collation{
		header: header,
		body:   []byte{1, 2, 3},
	}

	// We set the chunk root.
	collation.CalculateChunkRoot()

	if err := shard.SaveCollation(collation); err != nil {
		t.Fatalf("failed to save collation to shardDB: %v", err)
	}

	if err := shard.SetCanonical(header); err != nil {
		t.Fatalf("failed to set header as canonical: %v", err)
	}

	canonicalCollation, err := shard.CanonicalCollation(shardID, period)
	if err != nil {
		t.Fatalf("failed to get canonical collation from shardDB: %v", err)
	}
	if canonicalCollation.Hash() != collation.Hash() {
		t.Errorf("collations are not equal. want=%v. got=%v.", collation, canonicalCollation)
	}
}

func TestShard_BodyByChunkRoot(t *testing.T) {
	body := []byte{1, 2, 3, 4, 5}
	shardID := big.NewInt(1)
	shardDB := database.MakeShardKV()
	shard := MakeShard(shardID, shardDB)

	if err := shard.SaveBody(body); err != nil {
		t.Fatalf("cannot save body: %v", err)
	}

	// Right now this just hashes the body. We will instead need to
	// blob serialize.
	// TODO: blob serialization.
	chunkRoot := common.BytesToHash(body)

	dbBody, err := shard.BodyByChunkRoot(&chunkRoot)
	if err != nil {
		t.Errorf("cannot fetch body by chunk root: %v", err)
	}

	if !bytes.Equal(body, dbBody) {
		t.Errorf("bodies not equal. want=%v. got=%v", body, dbBody)
	}
}

func TestShard_CheckAvailability(t *testing.T) {
	shardID := big.NewInt(1)
	period := big.NewInt(1)
	proposerAddress := common.StringToAddress("")
	proposerSignature := []byte{}
	emptyHash := common.StringToHash("")
	header := NewCollationHeader(shardID, &emptyHash, period, &proposerAddress, proposerSignature)

	shardDB := database.MakeShardKV()
	shard := MakeShard(shardID, shardDB)

	collation := &Collation{
		header: header,
		body:   []byte{1, 2, 3},
	}

	// We set the chunk root.
	collation.CalculateChunkRoot()

	if err := shard.SaveBody(collation.body); err != nil {
		t.Fatalf("cannot save body: %v", err)
	}

	available, err := shard.CheckAvailability(header)
	if err != nil {
		t.Errorf("could not check availability: %v", err)
	}
	if !available {
		t.Errorf("collation body is not available: chunkRoot=%v, body=%v", header.ChunkRoot(), collation.body)
	}
}
