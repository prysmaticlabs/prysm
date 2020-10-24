package beaconv1

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

type stateContainer struct {
	state     *beaconstate.BeaconState
	blockRoot []byte
	stateRoot []byte
}

func fillDBTestState(ctx context.Context, t *testing.T, db db.Database) ([]*stateContainer, *stategen.State) {
	stateSumCache := stategen.New(db, cache.NewStateSummaryCache())
	beaconState, keys := testutil.DeterministicGenesisState(t, 64)
	require.NoError(t, db.SaveState(ctx, beaconState, params.BeaconConfig().ZeroHash))
	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, genesisBlk))
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisBlkRoot))

	count := uint64(130)
	blks := make([]*ethpb_alpha.SignedBeaconBlock, count)
	stateContainers := make([]*stateContainer, count)
	blks[0] = genesisBlk
	stateContainers[0] = &stateContainer{
		state:     beaconState.Copy(),
		blockRoot: genesisBlkRoot[:],
		stateRoot: stateRoot[:],
	}
	blockConf := &testutil.BlockGenConfig{
		NumAttestations: 1,
	}
	for i := uint64(1); i < count; i++ {
		b, err := testutil.GenerateFullBlock(beaconState, keys, blockConf, i-1)
		assert.NoError(t, err)
		beaconState, err = state.ExecuteStateTransition(ctx, beaconState, b)
		require.NoError(t, err)
		stateRoot, err := beaconState.HashTreeRoot(ctx)
		require.NoError(t, err)

		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		stateSum := &p2ppb.StateSummary{
			Slot: i,
			Root: root[:],
		}
		require.NoError(t, db.SaveStateSummary(ctx, stateSum))
		stateSumCache.SaveStateSummary(ctx, b, root)
		require.NoError(t, db.SaveState(ctx, beaconState, root))

		blks[i] = b
		stateContainers[i] = &stateContainer{state: beaconState.Copy(), blockRoot: root[:], stateRoot: stateRoot[:]}
		if i == 30 {
			duplicateBlock, ok := proto.Clone(b).(*ethpb_alpha.SignedBeaconBlock)
			require.Equal(t, ok, true)
			duplicateBlock.Block.Body.Graffiti = bytesutil.PadTo([]byte("duplicate"), 32)
			require.NoError(t, db.SaveBlock(ctx, duplicateBlock))
		}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))

	headState := stateContainers[len(stateContainers)-1].state

	// Save finalized state.
	finalizedSlot, err := helpers.StartSlot(headState.FinalizedCheckpoint().Epoch)
	require.NoError(t, err)
	finalizedState := stateContainers[finalizedSlot]
	stateSumCache.SaveFinalizedState(finalizedState.state.Slot(), bytesutil.ToBytes32(finalizedState.blockRoot), beaconState)
	finalChkpt := &ethpb_alpha.Checkpoint{
		Epoch: headState.FinalizedCheckpointEpoch(),
		Root:  finalizedState.blockRoot,
	}
	require.NoError(t, db.SaveFinalizedCheckpoint(ctx, finalChkpt))

	// Save justified state.
	justifiedSlot, err := helpers.StartSlot(headState.CurrentJustifiedCheckpoint().Epoch)
	require.NoError(t, err)
	justifiedState := stateContainers[justifiedSlot]
	justifiedChkpt := &ethpb_alpha.Checkpoint{
		Epoch: headState.CurrentJustifiedCheckpoint().Epoch,
		Root:  justifiedState.blockRoot,
	}
	require.NoError(t, db.SaveJustifiedCheckpoint(ctx, justifiedChkpt))

	headBlock := blks[len(blks)-1]
	headRoot, err := headBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveHeadBlockRoot(ctx, headRoot))
	return stateContainers, stateSumCache
}

func TestServer_GetGenesis(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()
	st := testutil.NewBeaconState()
	genValRoot := bytesutil.ToBytes32([]byte("I am root"))
	bs := &Server{
		BeaconDB:           db,
		GenesisTimeFetcher: &mock.ChainService{},
		ChainInfoFetcher: &mock.ChainService{
			State:          st,
			ValidatorsRoot: genValRoot,
		},
	}
	res, err := bs.GetGenesis(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, params.BeaconConfig().GenesisForkVersion, res.GenesisForkVersion)
	pUnix, err := ptypes.TimestampProto(time.Unix(0, 0))
	require.NoError(t, err)
	assert.Equal(t, true, res.GenesisTime.Equal(pUnix))
	assert.DeepEqual(t, genValRoot[:], res.GenesisValidatorsRoot)

	bs.GenesisTimeFetcher = &mock.ChainService{Genesis: time.Unix(10, 0)}
	res, err = bs.GetGenesis(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	pUnix, err = ptypes.TimestampProto(time.Unix(10, 0))
	require.NoError(t, err)
	assert.Equal(t, true, res.GenesisTime.Equal(pUnix))
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
			DB:    db,
			Root:  headState.blockRoot,
			Block: &ethpb_alpha.SignedBeaconBlock{Block: &ethpb_alpha.BeaconBlock{StateRoot: headState.stateRoot}},
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{
				Root:  headState.state.FinalizedCheckpoint().Root,
				Epoch: headState.state.FinalizedCheckpointEpoch(),
			},
			CurrentJustifiedCheckPoint: &ethpb_alpha.Checkpoint{
				Root:  headState.state.CurrentJustifiedCheckpoint().Root,
				Epoch: headState.state.CurrentJustifiedCheckpoint().Epoch,
			},
			PreviousJustifiedCheckPoint: &ethpb_alpha.Checkpoint{
				Root:  headState.state.PreviousJustifiedCheckpoint().Root,
				Epoch: headState.state.PreviousJustifiedCheckpoint().Epoch,
			},
			State: headState.state,
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
			want:    stateContainers[30].stateRoot,
		},
		{
			name:    "invalid string",
			stateId: []byte("lorem ipsum"),
			wantErr: true,
		},
		{
			name:    "root",
			stateId: stateContainers[20].blockRoot,
			want:    stateContainers[20].stateRoot,
		},
		{
			name:    "genesis",
			stateId: []byte("genesis"),
			want:    stateContainers[0].stateRoot,
		},
		{
			name:    "genesis root",
			stateId: stateContainers[0].blockRoot,
			want:    stateContainers[0].stateRoot,
		},
		{
			name:    "head",
			stateId: []byte("head"),
			want:    headState.stateRoot,
		},
		{
			name:    "justified",
			stateId: []byte("justified"),
			want:    stateContainers[96].stateRoot,
		},
		{
			name:    "finalized",
			stateId: []byte("finalized"),
			want:    stateContainers[64].stateRoot,
		},
		{
			name:    "no state",
			stateId: stateContainers[32].stateRoot,
			wantErr: true,
		},
		{
			name:    "future slot",
			stateId: []byte("200"),
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

			if !reflect.DeepEqual(stateRootResp.StateRoot, tt.want) {
				t.Errorf("Expected roots to equal, expected: %#x, received: %#x", tt.want, stateRootResp.StateRoot)
			}
		})
	}
}

func TestServer_GetStateFork(t *testing.T) {
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
			DB:   db,
			Root: headState.blockRoot,
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{
				Root:  headState.state.FinalizedCheckpoint().Root,
				Epoch: headState.state.FinalizedCheckpointEpoch(),
			},
			CurrentJustifiedCheckPoint: &ethpb_alpha.Checkpoint{
				Root:  headState.state.CurrentJustifiedCheckpoint().Root,
				Epoch: headState.state.CurrentJustifiedCheckpoint().Epoch,
			},
			PreviousJustifiedCheckPoint: &ethpb_alpha.Checkpoint{
				Root:  headState.state.PreviousJustifiedCheckpoint().Root,
				Epoch: headState.state.PreviousJustifiedCheckpoint().Epoch,
			},
			State: headState.state,
		},
		StateGen: sc,
	}

	tests := []struct {
		name    string
		stateId []byte
		want    *beaconstate.BeaconState
		wantErr bool
	}{
		{
			name:    "slot",
			stateId: []byte("30"),
			want:    stateContainers[30].state,
		},
		{
			name:    "invalid string",
			stateId: []byte("lorem ipsum"),
			wantErr: true,
		},
		{
			name:    "root",
			stateId: stateContainers[20].blockRoot,
			want:    stateContainers[20].state,
		},
		{
			name:    "genesis",
			stateId: []byte("genesis"),
			want:    stateContainers[0].state,
		},
		{
			name:    "genesis root",
			stateId: params.BeaconConfig().ZeroHash[:],
			want:    stateContainers[0].state,
		},
		{
			name:    "head",
			stateId: []byte("head"),
			want:    headState.state,
		},
		{
			name:    "justified",
			stateId: []byte("justified"),
			want:    stateContainers[96].state,
		},
		{
			name:    "finalized",
			stateId: []byte("finalized"),
			want:    stateContainers[64].state,
		},
		{
			name:    "no state",
			stateId: stateContainers[20].stateRoot,
			wantErr: true,
		},
		{
			name:    "future slot",
			stateId: []byte("200"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forkResp, err := bs.GetStateFork(ctx, &ethpb.StateRequest{
				StateId: tt.stateId,
			})
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.NotEqual(t, err, nil)
				return
			}

			compareBytes(t, "previous version", tt.want.Fork().PreviousVersion, forkResp.Fork.PreviousVersion)
			compareBytes(t, "current version", tt.want.Fork().CurrentVersion, forkResp.Fork.CurrentVersion)
			if tt.want.Fork().Epoch != forkResp.Fork.Epoch {
				t.Errorf("Expected epoch to be: %d, received: %v", tt.want.Fork().Epoch, forkResp.Fork.Epoch)
			}
		})
	}
}

func TestServer_GetFinalityCheckpoints(t *testing.T) {
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
			DB:   db,
			Root: headState.blockRoot,
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{
				Root:  headState.state.FinalizedCheckpoint().Root,
				Epoch: headState.state.FinalizedCheckpointEpoch(),
			},
			CurrentJustifiedCheckPoint: &ethpb_alpha.Checkpoint{
				Root:  headState.state.CurrentJustifiedCheckpoint().Root,
				Epoch: headState.state.CurrentJustifiedCheckpoint().Epoch,
			},
			PreviousJustifiedCheckPoint: &ethpb_alpha.Checkpoint{
				Root:  headState.state.PreviousJustifiedCheckpoint().Root,
				Epoch: headState.state.PreviousJustifiedCheckpoint().Epoch,
			},
			State: headState.state,
		},
		StateGen: sc,
	}

	tests := []struct {
		name    string
		stateId []byte
		want    *beaconstate.BeaconState
		wantErr bool
	}{
		{
			name:    "slot",
			stateId: []byte("30"),
			want:    stateContainers[30].state,
		},
		{
			name:    "invalid string",
			stateId: []byte("lorem ipsum"),
			wantErr: true,
		},
		{
			name:    "root",
			stateId: stateContainers[20].blockRoot,
			want:    stateContainers[20].state,
		},
		{
			name:    "genesis",
			stateId: []byte("genesis"),
			want:    stateContainers[0].state,
		},
		{
			name:    "genesis root",
			stateId: params.BeaconConfig().ZeroHash[:],
			want:    stateContainers[0].state,
		},
		{
			name:    "head",
			stateId: []byte("head"),
			want:    headState.state,
		},
		{
			name:    "justified",
			stateId: []byte("justified"),
			want:    stateContainers[96].state,
		},
		{
			name:    "finalized",
			stateId: []byte("finalized"),
			want:    stateContainers[64].state,
		},
		{
			name:    "no state",
			stateId: stateContainers[20].stateRoot,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finalityResp, err := bs.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
				StateId: tt.stateId,
			})
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.NotEqual(t, err, nil)
				return
			}

			compareBytes(t, "finalized roots", tt.want.FinalizedCheckpoint().Root, finalityResp.Finalized.Root)
			compareBytes(t, "current justified roots", tt.want.CurrentJustifiedCheckpoint().Root, finalityResp.CurrentJustified.Root)
			compareBytes(t, "previous justified roots", tt.want.PreviousJustifiedCheckpoint().Root, finalityResp.PreviousJustified.Root)
		})
	}
}

func compareBytes(t *testing.T, name string, expected []byte, received []byte) {
	if !bytes.Equal(expected, received) {
		t.Errorf("Expected %s to equal, expected: %#x, received: %#x", name, expected, received)
	}
}
