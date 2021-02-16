package blockchain

import (
	"context"
	"sync"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	blockchainTesting "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_ReceiveBlock(t *testing.T) {
	ctx := context.Background()

	genesis, keys := testutil.DeterministicGenesisState(t, 64)
	genFullBlock := func(t *testing.T, conf *testutil.BlockGenConfig, slot types.Slot) *ethpb.SignedBeaconBlock {
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
		name      string
		args      args
		wantedErr string
		check     func(*testing.T, *Service)
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
				pending := s.exitPool.PendingExits(genesis, 1, true /* no limit */)
				if len(pending) != 0 {
					t.Errorf(
						"Did not mark the correct number of exits. Got %d pending but wanted %d",
						len(pending),
						0,
					)
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
			beaconDB := testDB.SetupDB(t)
			genesisBlockRoot := bytesutil.ToBytes32(nil)
			require.NoError(t, beaconDB.SaveState(ctx, genesis, genesisBlockRoot))

			cfg := &Config{
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
			s, err := NewService(ctx, cfg)
			require.NoError(t, err)
			require.NoError(t, s.saveGenesisData(ctx, genesis))
			gBlk, err := s.beaconDB.GenesisBlock(ctx)
			require.NoError(t, err)
			gRoot, err := gBlk.Block.HashTreeRoot()
			require.NoError(t, err)
			s.finalizedCheckpt = &ethpb.Checkpoint{Root: gRoot[:]}
			root, err := tt.args.block.Block.HashTreeRoot()
			require.NoError(t, err)
			err = s.ReceiveBlock(ctx, tt.args.block, root)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
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
	beaconDB := testDB.SetupDB(t)
	genesisBlockRoot := bytesutil.ToBytes32(nil)
	require.NoError(t, beaconDB.SaveState(ctx, genesis, genesisBlockRoot))
	cfg := &Config{
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
	s, err := NewService(ctx, cfg)
	require.NoError(t, err)
	require.NoError(t, s.saveGenesisData(ctx, genesis))
	gBlk, err := s.beaconDB.GenesisBlock(ctx)
	require.NoError(t, err)
	gRoot, err := gBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	s.finalizedCheckpt = &ethpb.Checkpoint{Root: gRoot[:]}
	root, err := b.Block.HashTreeRoot()
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

func TestService_ReceiveBlockBatch(t *testing.T) {
	ctx := context.Background()

	genesis, keys := testutil.DeterministicGenesisState(t, 64)
	genFullBlock := func(t *testing.T, conf *testutil.BlockGenConfig, slot types.Slot) *ethpb.SignedBeaconBlock {
		blk, err := testutil.GenerateFullBlock(genesis, keys, conf, slot)
		assert.NoError(t, err)
		return blk
	}

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
			name: "applies block with state transition",
			args: args{
				block: genFullBlock(t, testutil.DefaultBlockGenConfig(), 2 /*slot*/),
			},
			check: func(t *testing.T, s *Service) {
				assert.Equal(t, types.Slot(2), s.head.state.Slot(), "Incorrect head state slot")
				assert.Equal(t, types.Slot(2), s.head.block.Block.Slot, "Incorrect head block slot")
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
			beaconDB := testDB.SetupDB(t)
			genesisBlockRoot, err := genesis.HashTreeRoot(ctx)
			require.NoError(t, err)
			cfg := &Config{
				BeaconDB: beaconDB,
				ForkChoiceStore: protoarray.New(
					0, // justifiedEpoch
					0, // finalizedEpoch
					genesisBlockRoot,
				),
				StateNotifier: &blockchainTesting.MockStateNotifier{RecordEvents: true},
				StateGen:      stategen.New(beaconDB),
			}
			s, err := NewService(ctx, cfg)
			require.NoError(t, err)
			err = s.saveGenesisData(ctx, genesis)
			require.NoError(t, err)
			gBlk, err := s.beaconDB.GenesisBlock(ctx)
			require.NoError(t, err)

			gRoot, err := gBlk.Block.HashTreeRoot()
			require.NoError(t, err)
			s.finalizedCheckpt = &ethpb.Checkpoint{Root: gRoot[:]}
			root, err := tt.args.block.Block.HashTreeRoot()
			require.NoError(t, err)
			blks := []*ethpb.SignedBeaconBlock{tt.args.block}
			roots := [][32]byte{root}
			err = s.ReceiveBlockBatch(ctx, blks, roots)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
				tt.check(t, s)
			}
		})
	}
}

func TestService_HasInitSyncBlock(t *testing.T) {
	s, err := NewService(context.Background(), &Config{StateNotifier: &blockchainTesting.MockStateNotifier{}})
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

func TestCheckSaveHotStateDB_Enabling(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	hook := logTest.NewGlobal()
	s, err := NewService(context.Background(), &Config{StateGen: stategen.New(beaconDB)})
	require.NoError(t, err)
	st := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(epochsSinceFinalitySaveHotStateDB))
	s.genesisTime = time.Now().Add(time.Duration(-1*int64(st)*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second)
	s.finalizedCheckpt = &ethpb.Checkpoint{}

	require.NoError(t, s.checkSaveHotStateDB(context.Background()))
	assert.LogsContain(t, hook, "Entering mode to save hot states in DB")
}

func TestCheckSaveHotStateDB_Disabling(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	hook := logTest.NewGlobal()
	s, err := NewService(context.Background(), &Config{StateGen: stategen.New(beaconDB)})
	require.NoError(t, err)
	s.finalizedCheckpt = &ethpb.Checkpoint{}
	require.NoError(t, s.checkSaveHotStateDB(context.Background()))
	s.genesisTime = time.Now()

	require.NoError(t, s.checkSaveHotStateDB(context.Background()))
	assert.LogsContain(t, hook, "Exiting mode to save hot states in DB")
}

func TestCheckSaveHotStateDB_Overflow(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	hook := logTest.NewGlobal()
	s, err := NewService(context.Background(), &Config{StateGen: stategen.New(beaconDB)})
	require.NoError(t, err)
	s.finalizedCheckpt = &ethpb.Checkpoint{Epoch: 10000000}
	s.genesisTime = time.Now()

	require.NoError(t, s.checkSaveHotStateDB(context.Background()))
	assert.LogsDoNotContain(t, hook, "Entering mode to save hot states in DB")
}
