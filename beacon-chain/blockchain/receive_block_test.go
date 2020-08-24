package blockchain

import (
	"context"
	"sync"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	blockchainTesting "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_ReceiveBlock(t *testing.T) {
	ctx := context.Background()

	genesis, keys := testutil.DeterministicGenesisState(t, 64)
	genFullBlock := func(t *testing.T, conf *testutil.BlockGenConfig, slot uint64) *ethpb.SignedBeaconBlock {
		blk, err := testutil.GenerateFullBlock(genesis, keys, conf, slot)
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
		name    string
		args    args
		wantErr bool
		check   func(*testing.T, *Service)
	}{
		{
			name: "applies block with state transition",
			args: args{
				block: genFullBlock(t, testutil.DefaultBlockGenConfig(), 2 /*slot*/),
			},
			check: func(t *testing.T, s *Service) {
				if hs := s.head.state.Slot(); hs != 2 {
					t.Errorf("Unexpected state slot. Got %d but wanted %d", hs, 2)
				}
				if bs := s.head.block.Block.Slot; bs != 2 {
					t.Errorf("Unexpected head block slot. Got %d but wanted %d", bs, 2)
				}
			},
		},
		{
			name: "saves attestations to pool",
			args: args{
				block: genFullBlock(t,
					&testutil.BlockGenConfig{
						NumProposerSlashings: 0,
						NumAttesterSlashings: 0,
						NumAttestations:      2,
						NumDeposits:          0,
						NumVoluntaryExits:    0,
					},
					1, /*slot*/
				),
			},
			check: func(t *testing.T, s *Service) {
				if baCount := len(s.attPool.BlockAttestations()); baCount != 2 {
					t.Errorf("Did not get the correct number of block attestations saved to the pool. "+
						"Got %d but wanted %d", baCount, 2)
				}
			},
		},

		{
			name: "updates exit pool",
			args: args{
				block: genFullBlock(t, &testutil.BlockGenConfig{
					NumProposerSlashings: 0,
					NumAttesterSlashings: 0,
					NumAttestations:      0,
					NumDeposits:          0,
					NumVoluntaryExits:    3,
				},
					1, /*slot*/
				),
			},
			check: func(t *testing.T, s *Service) {
				var n int
				for i := uint64(0); int(i) < genesis.NumValidators(); i++ {
					if s.exitPool.HasBeenIncluded(i) {
						n++
					}
				}
				if n != 3 {
					t.Errorf("Did not mark the correct number of exits. Got %d but wanted %d", n, 3)
				}
			},
		},
		{
			name: "notifies block processed on state feed",
			args: args{
				block: genFullBlock(t, testutil.DefaultBlockGenConfig(), 1 /*slot*/),
			},
			check: func(t *testing.T, s *Service) {
				if recvd := len(s.stateNotifier.(*blockchainTesting.MockStateNotifier).ReceivedEvents()); recvd < 1 {
					t.Errorf("Received %d state notifications, expected at least 1", recvd)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, stateSummaryCache := testDB.SetupDB(t)
			genesisBlockRoot := bytesutil.ToBytes32(nil)
			require.NoError(t, db.SaveState(ctx, genesis, genesisBlockRoot))

			cfg := &Config{
				BeaconDB: db,
				ForkChoiceStore: protoarray.New(
					0, // justifiedEpoch
					0, // finalizedEpoch
					genesisBlockRoot,
				),
				AttPool:       attestations.NewPool(),
				ExitPool:      voluntaryexits.NewPool(),
				StateNotifier: &blockchainTesting.MockStateNotifier{RecordEvents: true},
				StateGen:      stategen.New(db, stateSummaryCache),
			}
			s, err := NewService(ctx, cfg)
			require.NoError(t, err)
			require.NoError(t, s.saveGenesisData(ctx, genesis))
			gBlk, err := s.beaconDB.GenesisBlock(ctx)
			require.NoError(t, err)
			gRoot, err := stateutil.BlockRoot(gBlk.Block)
			s.finalizedCheckpt = &ethpb.Checkpoint{Root: gRoot[:]}
			root, err := stateutil.BlockRoot(tt.args.block.Block)
			require.NoError(t, err)
			if err := s.ReceiveBlock(ctx, tt.args.block, root); (err != nil) != tt.wantErr {
				t.Errorf("ReceiveBlock() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				tt.check(t, s)
			}
		})
	}
}

func TestService_ReceiveBlockUpdateHead(t *testing.T) {
	ctx := context.Background()
	genesis, keys := testutil.DeterministicGenesisState(t, 64)
	b, err := testutil.GenerateFullBlock(genesis, keys, testutil.DefaultBlockGenConfig(), 1)
	assert.NoError(t, err)
	db, stateSummaryCache := testDB.SetupDB(t)
	genesisBlockRoot := bytesutil.ToBytes32(nil)
	require.NoError(t, db.SaveState(ctx, genesis, genesisBlockRoot))
	cfg := &Config{
		BeaconDB: db,
		ForkChoiceStore: protoarray.New(
			0, // justifiedEpoch
			0, // finalizedEpoch
			genesisBlockRoot,
		),
		AttPool:       attestations.NewPool(),
		ExitPool:      voluntaryexits.NewPool(),
		StateNotifier: &blockchainTesting.MockStateNotifier{RecordEvents: true},
		StateGen:      stategen.New(db, stateSummaryCache),
	}
	s, err := NewService(ctx, cfg)
	require.NoError(t, err)
	require.NoError(t, s.saveGenesisData(ctx, genesis))
	gBlk, err := s.beaconDB.GenesisBlock(ctx)
	require.NoError(t, err)
	gRoot, err := stateutil.BlockRoot(gBlk.Block)
	s.finalizedCheckpt = &ethpb.Checkpoint{Root: gRoot[:]}
	root, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		require.NoError(t, s.ReceiveBlock(ctx, b, root))
		wg.Done()
	}()
	wg.Wait()
	if recvd := len(s.stateNotifier.(*blockchainTesting.MockStateNotifier).ReceivedEvents()); recvd < 1 {
		t.Errorf("Received %d state notifications, expected at least 1", recvd)
	}
	// Verify fork choice has processed the block. (Genesis block and the new block)
	assert.Equal(t, 2, len(s.forkChoiceStore.Nodes()))
}

func TestService_ReceiveBlockInitialSync(t *testing.T) {
	ctx := context.Background()

	genesis, keys := testutil.DeterministicGenesisState(t, 64)
	genFullBlock := func(t *testing.T, conf *testutil.BlockGenConfig, slot uint64) *ethpb.SignedBeaconBlock {
		blk, err := testutil.GenerateFullBlock(genesis, keys, conf, slot)
		if err != nil {
			t.Error(err)
		}
		return blk
	}

	type args struct {
		block *ethpb.SignedBeaconBlock
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		check   func(*testing.T, *Service)
	}{
		{
			name: "applies block with state transition",
			args: args{
				block: genFullBlock(t, testutil.DefaultBlockGenConfig(), 2 /*slot*/),
			},
			check: func(t *testing.T, s *Service) {
				assert.Equal(t, uint64(2), s.head.state.Slot(), "Incorrect head state slot")
				assert.Equal(t, uint64(2), s.head.block.Block.Slot, "Incorrect head block slot")
			},
		},
		{
			name: "notifies block processed on state feed",
			args: args{
				block: genFullBlock(t, testutil.DefaultBlockGenConfig(), 1 /*slot*/),
			},
			check: func(t *testing.T, s *Service) {
				if recvd := len(s.stateNotifier.(*blockchainTesting.MockStateNotifier).ReceivedEvents()); recvd < 1 {
					t.Errorf("Received %d state notifications, expected at least 1", recvd)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, stateSummaryCache := testDB.SetupDB(t)
			genesisBlockRoot := bytesutil.ToBytes32(nil)

			cfg := &Config{
				BeaconDB: db,
				ForkChoiceStore: protoarray.New(
					0, // justifiedEpoch
					0, // finalizedEpoch
					genesisBlockRoot,
				),
				StateNotifier: &blockchainTesting.MockStateNotifier{RecordEvents: true},
				StateGen:      stategen.New(db, stateSummaryCache),
			}
			s, err := NewService(ctx, cfg)
			require.NoError(t, err)
			err = s.saveGenesisData(ctx, genesis)
			require.NoError(t, err)
			gBlk, err := s.beaconDB.GenesisBlock(ctx)
			require.NoError(t, err)

			gRoot, err := stateutil.BlockRoot(gBlk.Block)
			s.finalizedCheckpt = &ethpb.Checkpoint{Root: gRoot[:]}
			root, err := stateutil.BlockRoot(tt.args.block.Block)
			require.NoError(t, err)

			if err := s.ReceiveBlockInitialSync(ctx, tt.args.block, root); (err != nil) != tt.wantErr {
				t.Errorf("ReceiveBlockInitialSync() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				tt.check(t, s)
			}
		})
	}
}

func TestService_ReceiveBlockBatch(t *testing.T) {
	ctx := context.Background()

	genesis, keys := testutil.DeterministicGenesisState(t, 64)
	genFullBlock := func(t *testing.T, conf *testutil.BlockGenConfig, slot uint64) *ethpb.SignedBeaconBlock {
		blk, err := testutil.GenerateFullBlock(genesis, keys, conf, slot)
		if err != nil {
			t.Error(err)
		}
		return blk
	}

	type args struct {
		block *ethpb.SignedBeaconBlock
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		check   func(*testing.T, *Service)
	}{
		{
			name: "applies block with state transition",
			args: args{
				block: genFullBlock(t, testutil.DefaultBlockGenConfig(), 2 /*slot*/),
			},
			check: func(t *testing.T, s *Service) {
				assert.Equal(t, uint64(2), s.head.state.Slot(), "Incorrect head state slot")
				assert.Equal(t, uint64(2), s.head.block.Block.Slot, "Incorrect head block slot")
			},
		},
		{
			name: "notifies block processed on state feed",
			args: args{
				block: genFullBlock(t, testutil.DefaultBlockGenConfig(), 1 /*slot*/),
			},
			check: func(t *testing.T, s *Service) {
				if recvd := len(s.stateNotifier.(*blockchainTesting.MockStateNotifier).ReceivedEvents()); recvd < 1 {
					t.Errorf("Received %d state notifications, expected at least 1", recvd)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, stateSummaryCache := testDB.SetupDB(t)
			genesisBlockRoot, err := genesis.HashTreeRoot(ctx)
			require.NoError(t, err)
			cfg := &Config{
				BeaconDB: db,
				ForkChoiceStore: protoarray.New(
					0, // justifiedEpoch
					0, // finalizedEpoch
					genesisBlockRoot,
				),
				StateNotifier: &blockchainTesting.MockStateNotifier{RecordEvents: true},
				StateGen:      stategen.New(db, stateSummaryCache),
			}
			s, err := NewService(ctx, cfg)
			require.NoError(t, err)
			err = s.saveGenesisData(ctx, genesis)
			require.NoError(t, err)
			gBlk, err := s.beaconDB.GenesisBlock(ctx)
			require.NoError(t, err)

			gRoot, err := stateutil.BlockRoot(gBlk.Block)
			s.finalizedCheckpt = &ethpb.Checkpoint{Root: gRoot[:]}
			root, err := stateutil.BlockRoot(tt.args.block.Block)
			require.NoError(t, err)
			blks := []*ethpb.SignedBeaconBlock{tt.args.block}
			roots := [][32]byte{root}
			if err := s.ReceiveBlockBatch(ctx, blks, roots); (err != nil) != tt.wantErr {
				t.Errorf("ReceiveBlockBatch() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				tt.check(t, s)
			}
		})
	}
}

func TestService_HasInitSyncBlock(t *testing.T) {
	s, err := NewService(context.Background(), &Config{})
	require.NoError(t, err)
	r := [32]byte{'a'}
	if s.HasInitSyncBlock(r) {
		t.Error("Should not have block")
	}
	s.saveInitSyncBlock(r, testutil.NewBeaconBlock())
	if !s.HasInitSyncBlock(r) {
		t.Error("Should have block")
	}
}
