package protoarray

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestComputeDelta_ZeroHash(t *testing.T) {
	validatorCount := uint64(16)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, 0})
		oldBalances = append(oldBalances, 0)
		newBalances = append(newBalances, 0)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	if err != nil {
		t.Fatal(err)
	}
	if len(delta) != int(validatorCount) {
		t.Error("Incorrect length")
	}
	for _, d := range delta {
		if d != 0 {
			t.Error("Delta should be zero")
		}
	}
	for _, vote := range votes {
		if vote.currentRoot != vote.nextRoot {
			t.Errorf("The vote should have changed")
		}
	}
}

func TestComputeDelta_AllVoteTheSame(t *testing.T) {
	validatorCount := uint64(16)
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{params.BeaconConfig().ZeroHash, indexToHash(0), 0})
		oldBalances = append(oldBalances, balance)
		newBalances = append(newBalances, balance)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	if err != nil {
		t.Fatal(err)
	}
	if len(delta) != int(validatorCount) {
		t.Error("Incorrect length")
	}

	for i, d := range delta {
		if i == 0 {
			if uint64(d) != balance*validatorCount {
				t.Error("Did not get correct balance")
			}
		} else {
			if d != 0 {
				t.Error("Delta should be zero")
			}
		}
	}

	for _, vote := range votes {
		if vote.currentRoot != vote.nextRoot {
			t.Errorf("The vote should have changed")
		}
	}
}

func TestComputeDelta_DifferentVotes(t *testing.T) {
	validatorCount := uint64(16)
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{params.BeaconConfig().ZeroHash, indexToHash(i), 0})
		oldBalances = append(oldBalances, balance)
		newBalances = append(newBalances, balance)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	if err != nil {
		t.Fatal(err)
	}
	if len(delta) != int(validatorCount) {
		t.Error("Incorrect length")
	}

	for _, d := range delta {
		if uint64(d) != balance {
			t.Error("Did not get correct delta")
		}
	}

	for _, vote := range votes {
		if vote.currentRoot != vote.nextRoot {
			t.Errorf("The vote should have changed")
		}
	}
}

func TestComputeDelta_MovingVotes(t *testing.T) {
	validatorCount := uint64(16)
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	lastIndex := uint64(len(indices) - 1)
	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{indexToHash(0), indexToHash(lastIndex), 0})
		oldBalances = append(oldBalances, balance)
		newBalances = append(newBalances, balance)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	if err != nil {
		t.Fatal(err)
	}
	if len(delta) != int(validatorCount) {
		t.Error("Incorrect length")
	}

	for i, d := range delta {
		if i == 0 {
			if d != -int(balance*validatorCount) {
				t.Error("First root should have negative delta")
			}
		} else if i == int(lastIndex) {
			if d != int(balance*validatorCount) {
				t.Error("Last root should have positive delta")
			}
		} else {
			if d != 0 {
				t.Error("Delta should be zero")
			}
		}
	}

	for _, vote := range votes {
		if vote.currentRoot != vote.nextRoot {
			t.Errorf("The vote should have changed")
		}
	}
}

func TestComputeDelta_MoveOutOfTree(t *testing.T) {
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := []uint64{balance, balance}
	newBalances := []uint64{balance, balance}

	indices[indexToHash(1)] = 0

	votes = append(votes, Vote{indexToHash(1), params.BeaconConfig().ZeroHash, 0})
	votes = append(votes, Vote{indexToHash(1), [32]byte{'A'}, 0})

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	if err != nil {
		t.Fatal(err)
	}
	if len(delta) != 1 {
		t.Error("Incorrect length")
	}

	if delta[0] != 0-2*int(balance) {
		t.Error("Incorrect delta")
	}

	for _, vote := range votes {
		if vote.currentRoot != vote.nextRoot {
			t.Errorf("The vote should have changed")
		}
	}
}

func TestComputeDelta_ChangingBalances(t *testing.T) {
	oldBalance := uint64(32)
	newBalance := oldBalance * 2
	validatorCount := uint64(16)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := make([]uint64, 0)
	newBalances := make([]uint64, 0)

	indices[indexToHash(1)] = 0

	for i := uint64(0); i < validatorCount; i++ {
		indices[indexToHash(i)] = i
		votes = append(votes, Vote{indexToHash(0), indexToHash(1), 0})
		oldBalances = append(oldBalances, oldBalance)
		newBalances = append(newBalances, newBalance)
	}

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	if err != nil {
		t.Fatal(err)
	}
	if len(delta) != 16 {
		t.Error("Incorrect length")
	}
	for i, d := range delta {
		if i == 0 {
			if d != -int(oldBalance*validatorCount) {
				t.Error("First root should have negative delta")
			}
		} else if i == 1 {
			if d != int(newBalance*validatorCount) {
				t.Error("Last root should have positive delta")
			}
		} else {
			if d != 0 {
				t.Error("Delta should be zero")
			}
		}
	}

	for _, vote := range votes {
		if vote.currentRoot != vote.nextRoot {
			t.Errorf("The vote should have changed")
		}
	}
}

func TestComputeDelta_ValidatorAppear(t *testing.T) {
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := []uint64{balance}
	newBalances := []uint64{balance, balance}

	indices[indexToHash(1)] = 0
	indices[indexToHash(2)] = 1

	votes = append(votes, Vote{indexToHash(1), indexToHash(2), 0})
	votes = append(votes, Vote{indexToHash(1), indexToHash(2), 0})

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	if err != nil {
		t.Fatal(err)
	}

	if len(delta) != 2 {
		t.Error("Incorrect length")
	}

	if delta[0] != 0-int(balance) {
		t.Error("Incorrect delta")
	}
	if delta[1] != 2*int(balance) {
		t.Error("Incorrect delta")
	}

	for _, vote := range votes {
		if vote.currentRoot != vote.nextRoot {
			t.Errorf("The vote should have changed")
		}
	}
}

func TestComputeDelta_ValidatorDisappears(t *testing.T) {
	balance := uint64(32)
	indices := make(map[[32]byte]uint64)
	votes := make([]Vote, 0)
	oldBalances := []uint64{balance, balance}
	newBalances := []uint64{balance}

	indices[indexToHash(1)] = 0
	indices[indexToHash(2)] = 1

	votes = append(votes, Vote{indexToHash(1), indexToHash(2), 0})
	votes = append(votes, Vote{indexToHash(1), indexToHash(2), 0})

	delta, _, err := computeDeltas(context.Background(), indices, votes, oldBalances, newBalances)
	if err != nil {
		t.Fatal(err)
	}

	if len(delta) != 2 {
		t.Error("Incorrect length")
	}

	if delta[0] != 0-2*int(balance) {
		t.Error("Incorrect delta")
	}
	if delta[1] != int(balance) {
		t.Error("Incorrect delta")
	}

	for _, vote := range votes {
		if vote.currentRoot != vote.nextRoot {
			t.Errorf("The vote should have changed")
		}
	}
}

func indexToHash(i uint64) [32]byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], i)
	return hashutil.Hash(b[:])
}
