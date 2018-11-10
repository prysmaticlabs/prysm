package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"reflect"
	"testing"
)

func TestBlockVoteCacheReadWrite(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	blockVoteCache := utils.NewBlockVoteCache()
	blockVote1 := &utils.BlockVote{VoterIndices: []uint32{1, 2, 3}, VoteTotalDeposit: 6}
	blockVote2 := &utils.BlockVote{VoterIndices: []uint32{4, 5, 6}, VoteTotalDeposit: 15}
	blockHash1 := [32]byte{1}
	blockHash2 := [32]byte{2}
	blockVoteCache[blockHash1] = blockVote1
	blockVoteCache[blockHash2] = blockVote2

	var err error
	if err = db.WriteBlockVoteCache(blockVoteCache); err != nil {
		t.Errorf("failed to write block vote cache to DB")
	}

	blockVoteCache2, err := db.ReadBlockVoteCache([][32]byte{blockHash1, blockHash2})
	if err != nil {
		t.Errorf("failed to read block vote cache from DB")
	}

	reflect.DeepEqual(blockVoteCache, blockVoteCache2)
}
