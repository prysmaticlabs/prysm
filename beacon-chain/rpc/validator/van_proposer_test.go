package validator

import (
	"context"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"testing"
)

func TestProposer_UpdateStateRoot_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	testutil.ResetCache()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, genesis), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := &Server{
		BeaconDB:           db,
		HeadFetcher:        &mock.ChainService{State: beaconState, Root: parentRoot[:]},
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		BlockReceiver:      &mock.ChainService{},
		ChainStartFetcher:  &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		Eth1BlockFetcher:   &mockPOW.POWChain{},
		MockEth1Votes:      true,
		AttPool:            attestations.NewPool(),
		SlashingsPool:      slashings.NewPool(),
		ExitPool:           voluntaryexits.NewPool(),
		StateGen:           stategen.New(db),
		EnableVanguardNode: true,
	}

	randaoReveal, err := testutil.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	req := &ethpb.BlockRequest{
		Slot:         1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}

	proposerSlashings := make([]*ethpb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := types.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerSlashing, err := testutil.GenerateProposerSlashingForValidator(
			beaconState,
			privKeys[i],
			i, /* validator index */
		)
		require.NoError(t, err)
		proposerSlashings[i] = proposerSlashing
		err = proposerServer.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
		require.NoError(t, err)
	}

	attSlashings := make([]*ethpb.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
		attesterSlashing, err := testutil.GenerateAttesterSlashingForValidator(
			beaconState,
			privKeys[i+params.BeaconConfig().MaxProposerSlashings],
			types.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
		)
		require.NoError(t, err)
		attSlashings[i] = attesterSlashing
		err = proposerServer.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
		require.NoError(t, err)
	}
	block, err := proposerServer.GetBlock(ctx, req)
	require.NoError(t, err)
	prevStateRoot := block.StateRoot

	pandoraHeader, _ := testutil.NewPandoraBlock(block.Slot, 120)
	newBeaconBlock := testutil.NewBeaconBlockWithPandoraSharding(pandoraHeader, block.Slot)
	block.Body.PandoraShard = newBeaconBlock.Block.Body.PandoraShard

	block, err = proposerServer.UpdateStateRoot(ctx, block)
	require.NoError(t, err)
	assert.DeepNotEqual(t, prevStateRoot, block.StateRoot)
}
