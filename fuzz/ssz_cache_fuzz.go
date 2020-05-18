package fuzz

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// BeaconFuzzSSZCache for testing critical paths along the ssz cache for beacon state HTR.
func BeaconFuzzSSZCache(input []byte) {
	s := &pb.BeaconState{}
	if err := s.UnmarshalSSZ(input); err != nil {
		return
	}

	fc := featureconfig.Get()
	fc.EnableSSZCache = true
	featureconfig.Init(fc)

	a, err := stateutil.HashTreeRootState(s)
	if err != nil {
		return
	}

	fc.EnableSSZCache = false
	featureconfig.Init(fc)

	b, err := stateutil.HashTreeRootState(s)
	if err != nil {
		return
	}

	if a != b {
		panic("Cached and non cached hash tree root hashers produced different results")
	}
}
