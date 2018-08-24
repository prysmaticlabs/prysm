package casper

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// RotateValidatorSet is called every dynasty transition. The primary functions are:
// 1.) Go through queued validator indices and induct them to be active by setting start
// dynasty to current cycle.
// 2.) Remove bad active validator whose balance is below threshold to the exit set by
// setting end dynasty to current cycle.
func RotateValidatorSet(crystallized *types.CrystallizedState) {
	validators := crystallized.Validators()
	upperbound := len(ActiveValidatorIndices(crystallized))/30 + 1

	// Loop through active validator set, remove validator whose balance is below 50%.
	for _, index := range ActiveValidatorIndices(crystallized) {
		if validators[index].Balance < params.DefaultBalance/2 {
			validators[index].EndDynasty = crystallized.CurrentDynasty()
		}
	}
	// Get the total number of validator we can induct.
	inductNum := upperbound
	if len(QueuedValidatorIndices(crystallized)) < inductNum {
		inductNum = len(QueuedValidatorIndices(crystallized))
	}

	// Induct queued validator to active validator set until the switch dynasty is greater than current number.
	for _, index := range QueuedValidatorIndices(crystallized) {
		validators[index].StartDynasty = crystallized.CurrentDynasty()
		inductNum--
		if inductNum == 0 {
			break
		}
	}
}

// ActiveValidatorIndices filters out active validators based on start and end dynasty
// and returns their indices in a list.
func ActiveValidatorIndices(crystallized *types.CrystallizedState) []int {
	var indices []int
	validators := crystallized.Validators()
	dynasty := crystallized.CurrentDynasty()
	for i := 0; i < len(validators); i++ {
		if validators[i].StartDynasty <= dynasty && dynasty < validators[i].EndDynasty {
			indices = append(indices, i)
		}
	}
	return indices
}

// ExitedValidatorIndices filters out exited validators based on start and end dynasty
// and returns their indices in a list.
func ExitedValidatorIndices(crystallized *types.CrystallizedState) []int {
	var indices []int
	validators := crystallized.Validators()
	dynasty := crystallized.CurrentDynasty()
	for i := 0; i < len(validators); i++ {
		if validators[i].StartDynasty < dynasty && validators[i].EndDynasty < dynasty {
			indices = append(indices, i)
		}
	}
	return indices
}

// QueuedValidatorIndices filters out queued validators based on start and end dynasty
// and returns their indices in a list.
func QueuedValidatorIndices(crystallized *types.CrystallizedState) []int {
	var indices []int
	validators := crystallized.Validators()
	dynasty := crystallized.CurrentDynasty()
	for i := 0; i < len(validators); i++ {
		if validators[i].StartDynasty > dynasty {
			indices = append(indices, i)
		}
	}
	return indices
}

// SampleAttestersAndProposers returns lists of random sampled attesters and proposer indices.
func SampleAttestersAndProposers(seed common.Hash, crystallized *types.CrystallizedState) ([]int, int, error) {
	attesterCount := params.MinCommiteeSize
	if crystallized.ValidatorsLength() < params.MinCommiteeSize {
		attesterCount = crystallized.ValidatorsLength()
	}
	fmt.Println()
	indices, err := utils.ShuffleIndices(seed, ActiveValidatorIndices(crystallized))
	if err != nil {
		return nil, -1, err
	}
	return indices[:int(attesterCount)], indices[len(indices)-1], nil
}

// GetAttestersTotalDeposit from the pending attestations.
func GetAttestersTotalDeposit(active *types.ActiveState) uint64 {
	var numOfBits int
	for _, attestation := range active.PendingAttestations() {
		for _, byte := range attestation.AttesterBitfield {
			numOfBits += int(utils.BitSetCount(byte))
		}
	}
	// Assume there's no slashing condition, the following logic will change later phase.
	return uint64(numOfBits) * params.DefaultBalance
}

// GetIndicesForHeight returns the attester set of a given height.
func GetIndicesForHeight(crystallized *types.CrystallizedState, height uint64) (*pb.ShardAndCommitteeArray, error) {
	lcs := crystallized.LastStateRecalc()
	if !(lcs <= height && height < lcs+params.CycleLength*2) {
		return nil, fmt.Errorf("can not return attester set of given height, input height %v has to be in between %v and %v", height, lcs, lcs+params.CycleLength*2)
	}
	return crystallized.IndicesForHeights()[height-lcs], nil
}
