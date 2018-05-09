package sharding

import (
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

// Hash returns the hash of a collation's entire contents. Useful for comparison tests.
func (c *Collation) Hash() (hash common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, c)
	hw.Sum(hash[:0])
	return hash
}
func TestShard_ValidateShardID(t *testing.T) {
	header := &CollationHeader{shardID: big.NewInt(4)}
	shardDB := makeShardKV()
	shard := MakeShard(big.NewInt(3), shardDB)

	if err := shard.ValidateShardID(header); err == nil {
		t.Errorf("ShardID validation incorrect. Function should throw error when ShardID's do not match. want=%d. got=%d", header.ShardID().Int64(), shard.ShardID().Int64())
	}

	header2 := &CollationHeader{shardID: big.NewInt(100)}
	shard2 := MakeShard(big.NewInt(100), shardDB)

	if err := shard2.ValidateShardID(header2); err != nil {
		t.Errorf("ShardID validation incorrect. Function should not throw error when ShardID's match. want=%d. got=%d", header2.ShardID().Int64(), shard2.ShardID().Int64())
	}
}

func TestShard_HeaderByHash(t *testing.T) {
	root := common.StringToHash("hi")
	header := &CollationHeader{shardID: big.NewInt(1), chunkRoot: &root}
	shardDB := makeShardKV()
	shard := MakeShard(big.NewInt(1), shardDB)

	if err := shard.SaveHeader(header); err != nil {
		t.Fatalf("cannot save collation header: %v", err)
	}
	hash := header.Hash()

	dbHeader, err := shard.HeaderByHash(&hash)
	if err != nil {
		t.Fatalf("could not fetch collation header by hash: %v", err)
	}
	log.Printf("header in first test: %v", header.ChunkRoot().String())
	log.Printf("db header in first test: %v", dbHeader.ChunkRoot().String())
	// Compare the hashes.
	if header.Hash() != dbHeader.Hash() {
		t.Errorf("headers do not match. want=%v. got=%v", header, dbHeader)
	}
}

func TestShard_CollationByHash(t *testing.T) {
	collation := &Collation{
		header: &CollationHeader{shardID: big.NewInt(1)},
		body:   []byte{1, 2, 3},
	}
	shardDB := makeShardKV()
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
