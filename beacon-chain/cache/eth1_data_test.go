package cache

import (
	"strconv"
	"testing"
)

func TestEth1DataVoteKeyFn_OK(t *testing.T) {
	eInfo := &Eth1DataVote{
		VoteCount:    44,
		Eth1DataHash: [32]byte{'A'},
	}

	key, err := eth1DataVoteKeyFn(eInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != string(eInfo.Eth1DataHash[:]) {
		t.Errorf("Incorrect hash key: %s, expected %s", key, string(eInfo.Eth1DataHash[:]))
	}
}

func TestEth1DataVoteKeyFn_InvalidObj(t *testing.T) {
	_, err := eth1DataVoteKeyFn("bad")
	if err != ErrNotEth1DataVote {
		t.Errorf("Expected error %v, got %v", ErrNotEth1DataVote, err)
	}
}

func TestEth1DataVoteCache_CanAdd(t *testing.T) {
	cache := NewEth1DataVoteCache()

	eInfo := &Eth1DataVote{
		VoteCount:    55,
		Eth1DataHash: [32]byte{'B'},
	}
	count, err := cache.Eth1DataVote(eInfo.Eth1DataHash)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Error("Expected seed not to exist in empty cache")
	}

	if err := cache.AddEth1DataVote(eInfo); err != nil {
		t.Fatal(err)
	}
	count, err = cache.Eth1DataVote(eInfo.Eth1DataHash)
	if err != nil {
		t.Fatal(err)
	}
	if count != eInfo.VoteCount {
		t.Errorf(
			"Expected vote count to be %d, got %d",
			eInfo.VoteCount,
			count,
		)
	}
}

func TestEth1DataVoteCache_CanIncrement(t *testing.T) {
	cache := NewEth1DataVoteCache()

	eInfo := &Eth1DataVote{
		VoteCount:    55,
		Eth1DataHash: [32]byte{'B'},
	}

	if err := cache.AddEth1DataVote(eInfo); err != nil {
		t.Fatal(err)
	}

	_, err := cache.IncrementEth1DataVote(eInfo.Eth1DataHash)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = cache.IncrementEth1DataVote(eInfo.Eth1DataHash)
	count, _ := cache.IncrementEth1DataVote(eInfo.Eth1DataHash)

	if count != 58 {
		t.Errorf(
			"Expected vote count to be %d, got %d",
			58,
			count,
		)
	}
}

func TestEth1Data_MaxSize(t *testing.T) {
	cache := NewEth1DataVoteCache()

	for i := 0; i < maxEth1DataVoteSize+1; i++ {
		var hash [32]byte
		copy(hash[:], []byte(strconv.Itoa(i)))
		eInfo := &Eth1DataVote{
			Eth1DataHash: hash,
		}
		if err := cache.AddEth1DataVote(eInfo); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.eth1DataVoteCache.ListKeys()) != maxEth1DataVoteSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxEth1DataVoteSize,
			len(cache.eth1DataVoteCache.ListKeys()),
		)
	}
}
