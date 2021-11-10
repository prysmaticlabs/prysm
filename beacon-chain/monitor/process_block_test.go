package monitor

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestService_ProcessBlock(t *testing.T) {
	ctx := context.Background()

	genesis, keys := util.DeterministicGenesisState(t, 64)
	genFullBlock := func(t *testing.T, conf *util.BlockGenConfig, slot types.Slot) *ethpb.SignedBeaconBlock {
		blk, err := util.GenerateFullBlock(genesis, keys, conf, slot)
		assert.NoError(t, err)
		return blk
	}
	bc := params.BeaconConfig()
	bc.ShardCommitteePeriod = 0 // Required for voluntary exits test in reasonable time.
	params.OverrideBeaconConfig(bc)

	type args struct {
		block *ethpb.SignedBeaconBlock
	}
	tests := []struct {
		name      string
		args      args
		wantedErr string
		check     func(*testing.T, *Service)
	}{
		{
			name: "Logs block proposed by tracked validator",
			args: args{
				block: genFullBlock(t, util.DefaultBlockGenConfig(), 1 /*slot*/),
			},
			check: func(t *testing.T, s *Service) {
				if recvd := len(s.cfg.StateNotifier.(*blockchainTesting.MockStateNotifier).ReceivedEvents()); recvd < 1 {
					t.Errorf("Received %d state notifications, expected at least 1", recvd)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconDB := testDB.SetupDB(t)
			genesisBlockRoot := bytesutil.ToBytes32(nil)
			require.NoError(t, beaconDB.SaveState(ctx, genesis, genesisBlockRoot))

			cfg := &config{
				BeaconDB: beaconDB,
				ForkChoiceStore: protoarray.New(
					0, // justifiedEpoch
					0, // finalizedEpoch
					genesisBlockRoot,
				),
				AttPool:       attestations.NewPool(),
				ExitPool:      voluntaryexits.NewPool(),
				StateNotifier: &blockchainTesting.MockStateNotifier{RecordEvents: true},
				StateGen:      stategen.New(beaconDB),
			}
			s, err := NewService(ctx)
			require.NoError(t, err)
			s.cfg = cfg
			require.NoError(t, s.saveGenesisData(ctx, genesis))
			gBlk, err := s.cfg.BeaconDB.GenesisBlock(ctx)
			require.NoError(t, err)
			gRoot, err := gBlk.Block().HashTreeRoot()
			require.NoError(t, err)
			s.finalizedCheckpt = &ethpb.Checkpoint{Root: gRoot[:]}
			root, err := tt.args.block.Block.HashTreeRoot()
			require.NoError(t, err)
			err = s.ReceiveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(tt.args.block), root)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
				tt.check(t, s)
			}
		})
	}

}
