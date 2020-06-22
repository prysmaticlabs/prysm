package blocks_test

import (
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func FakeDeposits(n uint64) []*ethpb.Eth1Data {
	deposits := make([]*ethpb.Eth1Data, n)
	for i := uint64(0); i < n; i++ {
		deposits[i] = &ethpb.Eth1Data{
			DepositCount: 1,
			DepositRoot:  []byte("root"),
		}
	}
	return deposits
}

func TestEth1DataHasEnoughSupport(t *testing.T) {
	tests := []struct {
		stateVotes         []*ethpb.Eth1Data
		data               *ethpb.Eth1Data
		hasSupport         bool
		votingPeriodLength uint64
	}{
		{
			stateVotes: FakeDeposits(4 * params.BeaconConfig().SlotsPerEpoch),
			data: &ethpb.Eth1Data{
				DepositCount: 1,
				DepositRoot:  []byte("root"),
			},
			hasSupport:         true,
			votingPeriodLength: 7,
		}, {
			stateVotes: FakeDeposits(4 * params.BeaconConfig().SlotsPerEpoch),
			data: &ethpb.Eth1Data{
				DepositCount: 1,
				DepositRoot:  []byte("root"),
			},
			hasSupport:         false,
			votingPeriodLength: 8,
		}, {
			stateVotes: FakeDeposits(4 * params.BeaconConfig().SlotsPerEpoch),
			data: &ethpb.Eth1Data{
				DepositCount: 1,
				DepositRoot:  []byte("root"),
			},
			hasSupport:         false,
			votingPeriodLength: 10,
		},
	}

	params.SetupTestConfigCleanup(t)
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			c := params.BeaconConfig()
			c.EpochsPerEth1VotingPeriod = tt.votingPeriodLength
			params.OverrideBeaconConfig(c)

			s, err := beaconstate.InitializeFromProto(&pb.BeaconState{
				Eth1DataVotes: tt.stateVotes,
			})
			if err != nil {
				t.Fatal(err)
			}
			result, err := blocks.Eth1DataHasEnoughSupport(s, tt.data)
			if err != nil {
				t.Fatal(err)
			}

			if result != tt.hasSupport {
				t.Errorf(
					"blocks.Eth1DataHasEnoughSupport(%+v) = %t, wanted %t",
					tt.data,
					result,
					tt.hasSupport,
				)
			}
		})
	}
}
