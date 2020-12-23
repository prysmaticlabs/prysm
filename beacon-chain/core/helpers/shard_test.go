package helpers

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestActiveShardCount(t *testing.T) {
	if ActiveShardCount() != params.BeaconConfig().InitialActiveShards {
		t.Fatal("Did not get correct active shard count")
	}
}

func TestUpdatedGasPrice(t *testing.T) {
	tests := []struct {
		prevGasPrice          uint64
		shardBlockLength      uint64
		finalGasPrice         uint64
		adjustmentCoefficient uint64
	}{
		{
			// Test max gas price is the upper bound.
			prevGasPrice:          params.BeaconConfig().MaxGasPrice + 1,
			shardBlockLength:      params.BeaconConfig().TargetShardBlockSize + 1,
			adjustmentCoefficient: 8,
			finalGasPrice:         params.BeaconConfig().MaxGasPrice,
		},
		{
			// Test min gas price is the lower bound.
			prevGasPrice:          0,
			shardBlockLength:      params.BeaconConfig().TargetShardBlockSize - 1,
			adjustmentCoefficient: 8,
			finalGasPrice:         params.BeaconConfig().MinGasPrice,
		},
		{
			// Test max gas price is the upper bound.
			prevGasPrice:          10000,
			shardBlockLength:      params.BeaconConfig().TargetShardBlockSize + 10000,
			adjustmentCoefficient: 8,
			finalGasPrice:         10047,
		},
		{
			// Test decreasing gas price.
			prevGasPrice:          100000000,
			shardBlockLength:      params.BeaconConfig().TargetShardBlockSize - 1,
			adjustmentCoefficient: 8,
			finalGasPrice:         99999953,
		},
	}

	for _, tt := range tests {
		if UpdatedGasPrice(tt.prevGasPrice, tt.shardBlockLength, tt.adjustmentCoefficient) != tt.finalGasPrice {
			t.Errorf("UpdatedGasPrice(%d, %d) = %d, wanted: %d", tt.prevGasPrice, tt.shardBlockLength,
				UpdatedGasPrice(tt.prevGasPrice, tt.shardBlockLength, tt.adjustmentCoefficient), tt.finalGasPrice)
		}
	}
}

func TestShardCommittee(t *testing.T) {
	shardCommitteeSizePerEpoch := uint64(4)
	beaconState, err := testState(shardCommitteeSizePerEpoch * params.BeaconConfig().SlotsPerEpoch)
	require.NoError(t, err)
	tests := []struct {
		epoch     uint64
		shard     uint64
		committee []uint64
	}{
		{
			epoch:     0,
			shard:     0,
			committee: []uint64{79, 125},
		},
		{
			epoch:     0,
			shard:     1,
			committee: []uint64{64, 0},
		},
		{
			epoch:     params.BeaconConfig().ShardCommitteePeriod,
			shard:     0,
			committee: []uint64{79, 125},
		},
	}

	for _, tt := range tests {
		ClearCache()
		committee, err := ShardCommittee(beaconState, tt.epoch, tt.shard)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(tt.committee, committee) {
			t.Errorf(
				"Result committee was an unexpected value. Wanted %d, got %d",
				tt.committee,
				committee,
			)
		}
	}
}

func testState(vCount uint64) (*stateTrie.BeaconState, error) {
	validators := make([]*ethpb.Validator, vCount)
	balances := make([]uint64, vCount)
	onlineCountdown := make([]uint64, vCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
		onlineCountdown[i] = 1
	}
	votedIndices := make([]uint64, 0)
	for i := uint64(0); i < params.BeaconConfig().MaxValidatorsPerCommittee; i++ {
		votedIndices = append(votedIndices, i)
	}
	return stateTrie.InitializeFromProto(&pb.BeaconState{
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		BlockRoots:  [][]byte{{'a'}, {'b'}, {'c'}},
		Balances:    balances,
	})
}
