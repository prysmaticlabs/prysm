package utils

import (
	"math"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
)

func TestFaultyShuffleIndices(t *testing.T) {
	if _, err := ShuffleIndices(common.Hash{'a'}, params.MaxValidators+1); err == nil {
		t.Error("Shuffle should have failed when validator count exceeds MaxValidators")
	}
}

func TestShuffleIndices(t *testing.T) {
	hash1 := common.BytesToHash([]byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'a', 'b', 'c', 'd', 'e', 'f', 'g'})
	hash2 := common.BytesToHash([]byte{'1', '2', '3', '4', '5', '6', '7', '1', '2', '3', '4', '5', '6', '7', '1', '2', '3', '4', '5', '6', '7', '1', '2', '3', '4', '5', '6', '7', '1', '2', '3', '4', '5', '6', '7'})

	list1, err := ShuffleIndices(hash1, 100)
	if err != nil {
		t.Errorf("Shuffle failed with: %v", err)
	}

	list2, err := ShuffleIndices(hash2, 100)
	if err != nil {
		t.Errorf("Shuffle failed with: %v", err)
	}
	if reflect.DeepEqual(list1, list2) {
		t.Errorf("2 shuffled lists shouldn't be equal")
	}
}

func TestCutOffValidatorSet(t *testing.T) {
	// Test scenario #1: Assume there's enough validators to fill in all the heights.
	validatorCount := params.EpochLength * params.MinCommiteeSize
	cutoffsValidators := GetCutoffs(validatorCount)

	// The length of cutoff list should be 65. Since there is 64 heights per epoch,
	// it means during every height, a new set of 128 validators will form a committee.
	expectedCount := int(math.Ceil(float64(validatorCount)/params.MinCommiteeSize)) + 1
	if len(cutoffsValidators) != expectedCount {
		t.Errorf("Incorrect count for cutoffs validator. Wanted: %v, Got: %v", expectedCount, len(cutoffsValidators))
	}

	// Verify each cutoff is an increment of MinCommiteeSize, it means 128 validators forms a
	// a committee and get to attest per height.
	count := 0
	for _, cutoff := range cutoffsValidators {
		if cutoff != count {
			t.Errorf("cutoffsValidators did not get 128 increment. Wanted: count, Got: %v", cutoff)
		}
		count += params.MinCommiteeSize
	}

	// Test scenario #2: Assume there's not enough validators to fill in all the heights.
	validatorCount = 1000
	cutoffsValidators = unique(GetCutoffs(validatorCount))
	// With 1000 validators, we can't attest every height. Given min committee size is 128,
	// we can only attest 7 heights. round down 1000 / 128 equals to 7, means the length is 8.
	expectedCount = int(math.Ceil(float64(validatorCount) / params.MinCommiteeSize))
	if len(unique(cutoffsValidators)) != expectedCount {
		t.Errorf("Incorrect count for cutoffs validator. Wanted: %v, Got: %v", expectedCount, validatorCount/params.MinCommiteeSize)
	}

	// Verify each cutoff is an increment of 142~143 (1000 / 7).
	count = 0
	for _, cutoff := range cutoffsValidators {
		num := count * validatorCount / (len(cutoffsValidators) - 1)
		if cutoff != num {
			t.Errorf("cutoffsValidators did not get correct increment. Wanted: %v, Got: %v", num, cutoff)
		}
		count++
	}
}

// helper function to remove duplicates in a int slice.
func unique(ints []int) []int {
	keys := make(map[int]bool)
	list := []int{}
	for _, int := range ints {
		if _, value := keys[int]; !value {
			keys[int] = true
			list = append(list, int)
		}
	}
	return list

}
