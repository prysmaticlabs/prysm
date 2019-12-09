package stateutil

import (
	"fmt"
	"testing"

	fuzz "github.com/google/gofuzz"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestStateRootCacheFuzz_1(t *testing.T) {
	fuzzStateRootCache(t, 0, 1)
}

func TestStateRootCacheFuzz_2(t *testing.T) {
	fuzzStateRootCache(t, 0, 2)
}

func TestStateRootCacheFuzz_10(t *testing.T) {
	fuzzStateRootCache(t, 0, 10)
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
			fmt.Println("Hashing without cache")
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
			fmt.Println("Hashing with cache")
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
