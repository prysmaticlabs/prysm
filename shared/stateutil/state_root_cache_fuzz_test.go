package stateutil

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

type Hasher interface {
	HashTreeRootState(state *ethereum_beacon_p2p_v1.BeaconState) ([32]byte, error)
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

