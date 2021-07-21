package validator

import (
	"context"
	"testing"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	rpctestutil "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProposer_GetBlock_OK(t *testing.T) {
	expectedBlock := &ethpb.BeaconBlock{}
	vs := &Server{
		BlockProducer: &rpctestutil.MockBlockProducer{Block: expectedBlock},
		SyncChecker:   &mockSync.Sync{IsSyncing: false},
	}
	block, err := vs.GetBlock(context.Background(), &ethpb.BlockRequest{})
	require.NoError(t, err)
	assert.Equal(t, expectedBlock, block)
}

func TestProposer_GetBlock_SyncNotReady(t *testing.T) {
	vs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}
	_, err := vs.GetBlock(context.Background(), &ethpb.BlockRequest{})
	assert.ErrorContains(t, "Syncing to latest head, not ready to respond", err)
}

func TestProposer_ProposeBlock_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	genesis := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

	numDeposits := uint64(64)
	beaconState, _ := testutil.DeterministicGenesisState(t, numDeposits)
	bsRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

	c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
	proposerServer := &Server{
		Eth1InfoFetcher: &mockPOW.POWChain{},
		BlockReceiver:   c,
		HeadFetcher:     c,
		BlockNotifier:   c.BlockNotifier(),
		P2P:             mockp2p.NewTestP2P(t),
	}
	req := testutil.NewBeaconBlock()
	req.Block.Slot = 5
	req.Block.ParentRoot = bsRoot[:]
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(req)))
	_, err = proposerServer.ProposeBlock(context.Background(), req)
	assert.NoError(t, err, "Could not propose block correctly")
}
