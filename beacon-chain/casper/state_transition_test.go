package casper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
)

func TestTallyVoteBalances(t *testing.T) {

	var validators []*pb.ValidatorRecord
	var blockHash [32]byte

	blockVoteCache := make(map[[32]byte]*utils.VoteCache)
	initialBalance := uint64(1e9)
	for i := 0; i < 1000; i++ {
		validator := &pb.ValidatorRecord{
			WithdrawalShard: 0,
			Balance:         initialBalance,
			Status:          uint64(params.Active)}

		validators = append(validators, validator)
	}

	voteCache := &utils.VoteCache{
		VoterIndices:     []uint32{20, 10},
		VoteTotalDeposit: 300,
	}
	copy(blockHash[:], []byte{'t', 'e', 's', 't', 'i', 'n', 'g'})

	blockVoteCache[blockHash] = voteCache

	zeroBalance, _ := TallyVoteBalances([32]byte{}, 10, blockVoteCache, validators, 2, true)

	if zeroBalance != 0 {
		t.Fatalf("votes have been calculated despite blockhash not existing in cache")
	}

	voteBalance, newValidators := TallyVoteBalances(blockHash, 10, blockVoteCache, validators, 2, true)
	if voteBalance != 300 {
		t.Fatalf("vote balances is not the amount expected %d", voteBalance)
	}
	if newValidators[20].Balance == initialBalance {
		t.Fatalf("validator balance not changed %d ", newValidators[20].Balance)
	}

	if newValidators[10].Balance == initialBalance {
		t.Errorf("validator balance not changed %d ", newValidators[10].Balance)
	}
}
