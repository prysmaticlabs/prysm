package beaconv1

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

type stateContainer struct {
	state *beaconstate.BeaconState
	root  []byte
}

func fillDBTestState(ctx context.Context, t *testing.T, db db.Database) ([]*stateContainer, *stategen.State) {
	stateSumCache := stategen.New(db, cache.NewStateSummaryCache())
	beaconState, keys := testutil.DeterministicGenesisState(t, 64)
	genesisBlockRoot := bytesutil.ToBytes32(nil)
	require.NoError(t, db.SaveState(ctx, beaconState, genesisBlockRoot))
	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, genesisBlk))
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlkRoot))

	count := uint64(100)
	blks := make([]*ethpb_alpha.SignedBeaconBlock, count)
	stateContainers := make([]*stateContainer, count)
	blks[0] = genesisBlk
	stateContainers[0] = &stateContainer{
		state: beaconState,
		root:  genesisBlkRoot[:],
	}
	for i := uint64(0); i < count-1; i++ {
		b, err := testutil.GenerateFullBlock(beaconState, keys, testutil.DefaultBlockGenConfig(), i)
		assert.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, b))
		beaconState, err = state.ExecuteStateTransition(ctx, beaconState, b)
		require.NoError(t, err)

		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i+1] = b
		stateContainers[i+1] = &stateContainer{state: beaconState, root: root[:]}
		if i == 32 {
			stateSum := &p2ppb.StateSummary{
				Slot: i,
				Root: root[:],
			}
			require.NoError(t, db.SaveStateSummary(ctx, stateSum))
			stateSumCache.SaveStateSummary(ctx, b, root)
			finalChkpt := &ethpb_alpha.Checkpoint{
				Epoch: helpers.SlotToEpoch(i),
				Root:  root[:],
			}
			require.NoError(t, db.SaveFinalizedCheckpoint(ctx, finalChkpt))
			require.NoError(t, db.SaveState(ctx, beaconState, root))
		} else if i == 64 {
			stateSum := &p2ppb.StateSummary{
				Slot: i,
				Root: root[:],
			}
			require.NoError(t, db.SaveStateSummary(ctx, stateSum))
			stateSumCache.SaveStateSummary(ctx, b, root)
			justifiedChkpt := &ethpb_alpha.Checkpoint{
				Epoch: helpers.SlotToEpoch(i),
				Root:  root[:],
			}
			require.NoError(t, db.SaveJustifiedCheckpoint(ctx, justifiedChkpt))
			require.NoError(t, db.SaveState(ctx, beaconState, root))
		}
	}

	require.NoError(t, db.SaveBlocks(ctx, blks))
	headBlock := blks[len(blks)-1]
	headRoot, err := headBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	summary := &p2ppb.StateSummary{
		Root: headRoot[:],
		Slot: uint64(len(blks) - 1),
	}
	require.NoError(t, db.SaveStateSummary(ctx, summary))
	stateSumCache.SaveStateSummary(ctx, headBlock, headRoot)
	require.NoError(t, db.SaveHeadBlockRoot(ctx, headRoot))
	return stateContainers, stateSumCache
}

func TestServer_GetStateRoot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	stateContainers, sc := fillDBTestState(ctx, t, db)
	headState := stateContainers[len(stateContainers)-1]
	bs := &Server{
		BeaconDB: db,
		GenesisTimeFetcher: &mock.ChainService{
			Genesis: time.Unix(0, 0),
		},
		ChainInfoFetcher: &mock.ChainService{
			DB:                         db,
			Root:                       headState.root,
			FinalizedCheckPoint:        &ethpb_alpha.Checkpoint{Root: stateContainers[32].root},
			CurrentJustifiedCheckPoint: &ethpb_alpha.Checkpoint{Root: stateContainers[64].root},
			State:                      headState.state,
		},
		StateGen: sc,
	}

	tests := []struct {
		name    string
		stateId []byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "slot",
			stateId: []byte("30"),
			want:    stateContainers[30].root,
		},
		{
			name:    "root",
			stateId: stateContainers[20].root,
			want:    stateContainers[20].root,
		},
		{
			name:    "genesis",
			stateId: []byte("genesis"),
			want:    stateContainers[0].root,
		},
		{
			name:    "genesis root",
			stateId: stateContainers[0].root,
			want:    stateContainers[0].root,
		},
		{
			name:    "head",
			stateId: []byte("head"),
			want:    headState.root,
		},
		{
			name:    "justified",
			stateId: []byte("justified"),
			want:    stateContainers[65].root,
		},
		{
			name:    "finalized",
			stateId: []byte("finalized"),
			want:    stateContainers[33].root,
		},
		{
			name:    "no state",
			stateId: []byte("105"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateRootResp, err := bs.GetStateRoot(ctx, &ethpb.StateRequest{
				StateId: tt.stateId,
			})
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.NotEqual(t, err, nil)
				return
			}

			if !reflect.DeepEqual(stateRootResp.StateRoot, tt.stateId) {
				t.Error("Expected roots to equal")
			}
		})
	}
}
