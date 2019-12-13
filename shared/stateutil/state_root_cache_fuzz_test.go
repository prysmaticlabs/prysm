package stateutil

import (
	"fmt"
	"strconv"
	"testing"

	fuzz "github.com/google/gofuzz"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestStateRootCacheFuzz_1(t *testing.T) {
	fuzzStateRootCache(t, 0, 1)
}

func TestStateRootCacheFuzz_4(t *testing.T) {
	fuzzStateRootCache(t, 0, 4)
}

func TestStateRootCacheFuzz_10(t *testing.T) {
	fuzzStateRootCache(t, 0, 10)
}

func TestStateRootCacheFuzz_50(t *testing.T) {
	fuzzStateRootCache(t, 0, 50)
}

func TestStateRootCacheFuzz_100(t *testing.T) {
	fuzzStateRootCache(t, 0, 100)
}

func TestStateRootCacheFuzz_1000(t *testing.T) {
	fuzzStateRootCache(t, 1, 1000)
}

func TestStateRootCacheFuzz_10000(t *testing.T) {
	fuzzStateRootCache(t, 2, 10000)
}

func fuzzStateRootCache(t *testing.T, seed int64, iterations uint64) {
	fuzzer := fuzz.NewWithSeed(seed)
	state := &ethereum_beacon_p2p_v1.BeaconState{}

	hasher := &stateRootHasher{}
	hasherWithCache := globalHasher

	mismatch := 0
	for i := uint64(0); i < iterations; i++ {
		fuzzer.Fuzz(state)
		var a, b [32]byte
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Non-cached HTR panicked on iteration %d", i)
					panic(r)
				}
			}()
			var err error
			fmt.Println("Without cache")
			a, err = hasher.hashTreeRootState(state)
			if err != nil {
				t.Fatal(err)
			}
		}()

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Cached HTR panicked on iteration %d", i)
					panic(r)
				}
			}()
			var err error
			fmt.Println("With cache")
			b, err = hasherWithCache.hashTreeRootState(state)
			if err != nil {
				t.Fatal(err)
			}
		}()

		if a != b {
			mismatch++
		}
	}
	if mismatch > 0 {
		t.Fatalf("%d of %d random states had different roots", mismatch, iterations)
	}
}

func TestHashTreeRootState_ElementsChanged_RecomputeBranch(t *testing.T) {
	hasher := &stateRootHasher{}
	hasherWithCache := globalHasher
	state := &ethereum_beacon_p2p_v1.BeaconState{}
	initialRoots := make([][]byte, 5)
	for i := 0; i < len(initialRoots); i++ {
		var someRt [32]byte
		copy(someRt[:], "hello")
		initialRoots[i] = someRt[:]
	}
	state.RandaoMixes = initialRoots
	if _, err := hasherWithCache.hashTreeRootState(state); err != nil {
		t.Fatal(err)
	}

	badRoots := make([][]byte, 5)
	for i := 0; i < len(badRoots); i++ {
		var someRt [32]byte
		copy(someRt[:], strconv.Itoa(i))
		badRoots[i] = someRt[:]
	}

	state.RandaoMixes = badRoots
	r1, err := hasher.hashTreeRootState(state)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := hasherWithCache.hashTreeRootState(state)
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Errorf("Wanted %#x (nocache), received %#x (withcache)", r1, r2)
	}
}
