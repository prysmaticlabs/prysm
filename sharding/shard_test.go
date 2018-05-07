package sharding

import (
	"math/big"
	"testing"
)

func TestShard_ValidateShardID(t *testing.T) {
	header := &CollationHeader{shardID: big.NewInt(4)}
	shard := MakeShard(big.NewInt(3))

	if err := shard.ValidateShardID(header); err == nil {
		t.Fatalf("Shard ID validation incorrect. Function should throw error when shardID's do not match. want=%d. got=%d", header.shardID.Int64(), shard.ShardID().Int64())
	}

	header2 := &CollationHeader{shardID: big.NewInt(100)}
	shard2 := MakeShard(big.NewInt(100))

	if err := shard2.ValidateShardID(header2); err != nil {
		t.Fatalf("Shard ID validation incorrect. Function should not throw error when shardID's match. want=%d. got=%d", header2.shardID.Int64(), shard2.ShardID().Int64())
	}
}
