package altair_test

import (
	"context"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/state-altair"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestGenesisBeaconState_1000(t *testing.T) {
	state.SkipSlotCache.Disable()
	defer state.SkipSlotCache.Enable()
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	deposits := make([]*ethpb.Deposit, 300000)
	var genesisTime uint64
	eth1Data := &ethpb.Eth1Data{}
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(&deposits)
		fuzzer.Fuzz(&genesisTime)
		fuzzer.Fuzz(eth1Data)
		gs, err := altair.GenesisBeaconState(context.Background(), deposits, genesisTime, eth1Data)
		if err != nil {
			if gs != nil {
				t.Fatalf("Genesis state should be nil on err. found: %v on error: %v for inputs deposit: %v "+
					"genesis time: %v eth1data: %v", gs, err, deposits, genesisTime, eth1Data)
			}
		}
	}
}

func TestOptimizedGenesisBeaconState_1000(t *testing.T) {
	state.SkipSlotCache.Disable()
	defer state.SkipSlotCache.Enable()
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	var genesisTime uint64
	preState := &stateAltair.BeaconState{}
	eth1Data := &ethpb.Eth1Data{}
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(&genesisTime)
		fuzzer.Fuzz(eth1Data)
		fuzzer.Fuzz(preState)
		gs, err := altair.OptimizedGenesisBeaconState(genesisTime, preState, eth1Data)
		if err != nil {
			if gs != nil {
				t.Fatalf("Genesis state should be nil on err. found: %v on error: %v for inputs genesis time: %v "+
					"pre state: %v eth1data: %v", gs, err, genesisTime, preState, eth1Data)
			}
		}
	}
}
