package blockchain

import (
	"context"
	"testing"
	"time"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func Test_IsOptimisticCandidateBlock(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	params.BeaconConfig().SafeSlotsToImportOptimistically = 128
	service.genesisTime = time.Now().Add(-time.Second * 12 * 2 * 128)

	tests := []struct {
		name      string
		blk       block.BeaconBlock
		justified block.SignedBeaconBlock
		want      bool
	}{
		{
			name: "deep block",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 1
				wr, err := wrapper.WrappedBellatrixBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				wr, err := wrapper.WrappedBellatrixSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			want: true,
		},
		{
			name: "shallow block, Altair justified chkpt",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockAltair()
				blk.Block.Slot = 200
				wr, err := wrapper.WrappedAltairBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockAltair()
				blk.Block.Slot = 32
				wr, err := wrapper.WrappedAltairSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			want: false,
		},
		{
			name: "shallow block, Bellatrix justified chkpt without execution",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 200
				wr, err := wrapper.WrappedBellatrixBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				wr, err := wrapper.WrappedBellatrixSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			want: false,
		},
		{
			name: "shallow block, execution enabled justified chkpt",
			blk: func(tt *testing.T) block.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 200
				wr, err := wrapper.WrappedBellatrixBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) block.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				blk.Block.Body.ExecutionPayload.ParentHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.FeeRecipient = bytesutil.PadTo([]byte{'a'}, fieldparams.FeeRecipientLength)
				blk.Block.Body.ExecutionPayload.StateRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.ReceiptsRoot = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.LogsBloom = bytesutil.PadTo([]byte{'a'}, fieldparams.LogsBloomLength)
				blk.Block.Body.ExecutionPayload.Random = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.BaseFeePerGas = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				blk.Block.Body.ExecutionPayload.BlockHash = bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength)
				wr, err := wrapper.WrappedBellatrixSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			want: true,
		},
	}
	for _, tt := range tests {
		jroot, err := tt.justified.Block().HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, tt.justified))
		service.store.SetJustifiedCheckpt(
			&ethpb.Checkpoint{
				Root:  jroot[:],
				Epoch: slots.ToEpoch(tt.justified.Block().Slot()),
			})
		candidate, err := service.isOptimisticCandidateBlock(ctx, tt.blk)
		require.NoError(t, err)
		require.Equal(t, tt.want, candidate, tt.name)
	}
}
