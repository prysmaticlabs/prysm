package precompute

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessRewardsAndPenaltiesPrecompute(t *testing.T) {
	helpers.ClearAllCaches()
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(2048)
	state := buildState(e+3, validatorCount)
	startShard := uint64(960)
	atts := make([]*pb.PendingAttestation, 3)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    startShard + uint64(i),
					DataRoot: []byte{'A'},
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	state.PreviousEpochAttestations = atts
	state.CurrentCrosslinks[startShard] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}
	state.CurrentCrosslinks[startShard+1] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}
	state.CurrentCrosslinks[startShard+2] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}

	vp, bp := New(context.Background(), state)
	vp, bp, err := ProcessAttestations(context.Background(), state, vp, bp)
	if err != nil {
		t.Fatal(err)
	}

	state, err = ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	if err != nil {
		t.Fatal(err)
	}

	// Indices that voted
	wanted := uint64(32000001776)
	if state.Balances[4] != wanted {
		t.Errorf("wanted balance: %d, got: %d",
			wanted, state.Balances[4])
	}

	// Indices that did not vote - lose money
	wanted = uint64(31999797616)
	if state.Balances[0] != wanted {
		t.Errorf("wanted balance: %d, got: %d",
			wanted, state.Balances[0])
	}
}

func buildState(slot uint64, validatorCount uint64) *pb.BeaconState {
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = params.BeaconConfig().ZeroHash[:]
	}
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}
	return &pb.BeaconState{
		Slot:                        slot,
		Balances:                    validatorBalances,
		Validators:                  validators,
		CurrentCrosslinks:           make([]*ethpb.Crosslink, params.BeaconConfig().ShardCount),
		RandaoMixes:                 make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:            make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CompactCommitteesRoots:      make([][]byte, params.BeaconConfig().EpochsPerSlashingsVector),
		Slashings:                   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		BlockRoots:                  make([][]byte, params.BeaconConfig().SlotsPerEpoch*10),
		FinalizedCheckpoint:         &ethpb.Checkpoint{},
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{},
	}
}
