package blocks_test

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
			require.NoError(t, err)
			result, err := blocks.Eth1DataHasEnoughSupport(s, tt.data)
			require.NoError(t, err)

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

func TestAreEth1DataEqual(t *testing.T) {
	type args struct {
		a *ethpb.Eth1Data
		b *ethpb.Eth1Data
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true when both are nil",
			args: args{
				a: nil,
				b: nil,
			},
			want: true,
		},
		{
			name: "false when only one is nil",
			args: args{
				a: nil,
				b: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
			},
			want: false,
		},
		{
			name: "true when real equality",
			args: args{
				a: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
				b: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
			},
			want: true,
		},
		{
			name: "false is field value differs",
			args: args{
				a: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
				b: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 64,
					BlockHash:    make([]byte, 32),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, blocks.AreEth1DataEqual(tt.args.a, tt.args.b))
		})
	}
}

func TestProcessEth1Data_SetsCorrectly(t *testing.T) {
	beaconState, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{},
	})
	require.NoError(t, err)

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte{2},
				BlockHash:   []byte{3},
			},
		},
	}

	period := params.BeaconConfig().EpochsPerEth1VotingPeriod * params.BeaconConfig().SlotsPerEpoch
	for i := uint64(0); i < period; i++ {
		beaconState, err = blocks.ProcessEth1DataInBlock(beaconState, block)
		require.NoError(t, err)
	}

	newETH1DataVotes := beaconState.Eth1DataVotes()
	if len(newETH1DataVotes) <= 1 {
		t.Error("Expected new ETH1 data votes to have length > 1")
	}
	if !proto.Equal(beaconState.Eth1Data(), beaconstate.CopyETH1Data(block.Body.Eth1Data)) {
		t.Errorf(
			"Expected latest eth1 data to have been set to %v, received %v",
			block.Body.Eth1Data,
			beaconState.Eth1Data(),
		)
	}
}
