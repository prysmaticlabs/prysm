package validator

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	mockBC "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

// Empty Block Chain - No Head
func Test_DetectDoppelganger_NoHead(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	// Set head state to nil
	chainService := &mockChain.ChainService{State: nil}
	vs := &Server{
		Ctx: ctx,
		ChainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		BeaconDB:      beaconDB,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}

	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: nil,
	}
	_, err := vs.DetectDoppelganger(ctx, req)
	assert.ErrorContains(t, "Doppelganger rpc service - Could not get head state", err)
}

// Target and Head epochs are close to each other.
func Test_DetectDoppelganger_TargetHeadClose(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	priv1, err := bls.RandKey()
	require.NoError(t, err)

	pubKey1 := priv1.PublicKey().Marshal()
	slot := types.Slot(400)
	beaconState := &pbp2p.BeaconState{
		Slot: slot,
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch:       0,
				ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
				PublicKey:             pubKey1,
				WithdrawalCredentials: make([]byte, 32),
			},
		},
	}

	block := testutil.NewBeaconBlock()
	genesisRoot, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	trie, err := stateV0.InitializeFromProtoUnsafe(beaconState)
	require.NoError(t, err)
	vs := &Server{
		BeaconDB:           beaconDB,
		Ctx:                ctx,
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		HeadFetcher:        &mockChain.ChainService{State: trie, Root: genesisRoot[:]},
		StateGen:           stategen.New(beaconDB),
	}

	pKT := make([]*ethpb.PubKeyTarget, 0)

	// Use the same slot so that Head - Target is less than N(=2) Epochs apart.
	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: pubKey1,
		TargetEpoch: types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch)})
	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	res, err := vs.DetectDoppelganger(ctx, req)
	assert.NoError(t, err)
	// Nil Public Key(no duplicate retrieved because Target close to Head).
	assert.DeepEqual(t, []byte(nil), res.PublicKey)

	// No prevState
	pKT = make([]*ethpb.PubKeyTarget, 0)
	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: pubKey1,
		TargetEpoch: types.Epoch(slot.Sub(20) / params.BeaconConfig().SlotsPerEpoch)})
	req = &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	_, err = vs.DetectDoppelganger(ctx, req)
	// No Previous state is available
	assert.ErrorContains(t, "Doppelganger rpc service - Could not get previous state root", err)

	// Add a prevState
	/*
		prevState, newSignedblock, _, _, _ := createFullBlockWithOperations(t)
		newState, err := state.ProcessBlock(context.Background(), prevState, interfaces.WrappedPhase0SignedBeaconBlock(newSignedblock))
		require.NoError(t, err)
		require.NoError(t, newState.SetSlot(200))
		newRoot,err := newSignedblock.Block.HashTreeRoot()
		require.NoError(t, err)
		vs.HeadFetcher = &mockChain.ChainService{State: newState, Root: newRoot[:]} // newRoot[:]
		require.NoError(t, beaconDB.SaveStateSummary(ctx, &pbp2p.StateSummary{Root: newRoot[:]}))

		//require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, bytesutil.ToBytes32(newRoot)))
		err = vs.StateGen.SaveState(ctx,newRoot,newState)  //bytesutil.ToBytes32(
		require.NoError(t,err)

		//service := stategen.New(beaconDB)
		pState, _ := testutil.DeterministicGenesisState(t, 32)
		require.NoError(t, pState.SetSlot(420))
		blk := testutil.NewBeaconBlock()
		blkRoot, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveStateSummary(ctx, &pbp2p.StateSummary{Root: blkRoot[:]}))

		//vs.StateGen.SaveFinalizedState(slot,bytesutil.ToBytes32(newSignedblock.Block.ParentRoot), newState)

		//service := stategen.New(beaconDB)
		//err = service.SaveState(ctx,newRoot,newState)
		//require.NoError(t, err)
		//service.hotStateCache.put(bytesutil.ToBytes32(b1.Block.ParentRoot), genesis)
		_, err = vs.DetectDoppelganger(ctx, req)
		require.NoError(t, err)

	*/
}

// EXPERIMENTATIONs from here on
/*Unexpected error: could not process epoch with optimizations: failed to initialize precompute: nil validators in state
 */
func Test_DetectDoppelganger_NoPrevState0(t *testing.T) {

	prevConfig := params.BeaconConfig().Copy()
	defer params.OverrideBeaconConfig(prevConfig)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service := stategen.New(beaconDB)

	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	genesisStateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(genesis)))
	gRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	//require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Slot: 0, Root: gRoot[:]}))
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &pbp2p.StateSummary{Slot: 400, Root: gRoot[:]}))
	service.SaveFinalizedState(types.Slot(400), gRoot, beaconState)

	// This tests where hot state was already cached.
	_, err = service.StateByRoot(ctx, gRoot)
	require.NoError(t, err)

	pubKey1 := privKeys[0].PublicKey().Marshal()

	slot := types.Slot(400)

	vs := &Server{
		BeaconDB:           beaconDB,
		Ctx:                ctx,
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		HeadFetcher:        &mockChain.ChainService{State: beaconState, Root: gRoot[:]},
		StateGen:           service,
	}

	pKT := make([]*ethpb.PubKeyTarget, 0)
	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: pubKey1,
		TargetEpoch: types.Epoch(slot.Sub(20) / params.BeaconConfig().SlotsPerEpoch)})
	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	_, err = vs.DetectDoppelganger(ctx, req)
	require.NoError(t, err)

}
/*
func Test_DetectDoppelganger_NoPrevState2(t *testing.T){
	prevConfig := params.BeaconConfig().Copy()
	defer params.OverrideBeaconConfig(prevConfig)
	params.OverrideBeaconConfig(params.MainnetConfig())
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	assert.NoError(t, beaconDB.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(genesis)))
	validGenesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	genState, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(ctx, genState.Copy(), validGenesisRoot))
	roots, err, privKeys,st:= blockTree1(t, beaconDB, validGenesisRoot[:])
	require.NoError(t, err)
	random := testutil.NewBeaconBlock()
	random.Block.Slot = 101
	random.Block.ParentRoot = validGenesisRoot[:]
	assert.NoError(t, beaconDB.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(random)))
	randomParentRoot, err := random.Block.HashTreeRoot()
	assert.NoError(t, err)
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &pbp2p.StateSummary{Slot: genState.Slot(), Root: randomParentRoot[:]}))
	require.NoError(t, beaconDB.SaveState(ctx, genState.Copy(), randomParentRoot))
	err = genState.SetLatestBlockHeader(testutil.HydrateBeaconHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesis.Block.Slot,
		ParentRoot: genesis.Block.ParentRoot,
		BodyRoot:   validGenesisRoot[:],
	}))
	assert.NoError(t, err)

	randomParentRoot2 := roots[1]
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &pbp2p.StateSummary{Slot: st.Slot(), Root: randomParentRoot2}))
	require.NoError(t, beaconDB.SaveState(ctx, st.Copy(), bytesutil.ToBytes32(randomParentRoot2)))
	err = st.SetLatestBlockHeader(testutil.HydrateBeaconHeader(&ethpb.BeaconBlockHeader{
		Slot:       101 ,
		ParentRoot: validGenesisRoot[:],
		BodyRoot:   randomParentRoot2,
	}))
	assert.NoError(t, err)
	vs := &Server{
		BeaconDB:        beaconDB,
		StateGen:        stategen.New(beaconDB),
		//ForkChoiceStore: protoarray.New(0, 0, [32]byte{}),
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		HeadFetcher:        &mockChain.ChainService{State: st, Root: validGenesisRoot[:]},
	}
	pKT := make([]*ethpb.PubKeyTarget, 0)
	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: privKeys[0].PublicKey().Marshal(),
		TargetEpoch: types.Epoch(50 / params.BeaconConfig().SlotsPerEpoch)})
	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	_, err = vs.DetectDoppelganger(ctx, req)
	require.NoError(t, err)

}
*/
/*
func Test_DetectDoppelganger_NoPrevState1(t *testing.T) {
	prevConfig := params.BeaconConfig().Copy()
	defer params.OverrideBeaconConfig(prevConfig)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	service :=  stategen.New(beaconDB)

	beaconState, _, privKeys, blkContainers := fillDBTestFullBlocksWithOperations(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]
	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 100
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 100
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(b3)))
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &pbp2p.StateSummary{Slot: 100, Root: headBlock.BlockRoot}))


	vs := &Server{
		BeaconDB: beaconDB,
		HeadFetcher: &mockChain.ChainService{
			DB:                  beaconDB,
			State:               beaconState,
			Block:               interfaces.WrappedPhase0SignedBeaconBlock(headBlock.Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpb.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
		StateGen:           service,
		//HeadFetcher:        &mockChain.ChainService{State: beaconState, Root: gRoot[:]},
		Ctx:                ctx,
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
	}
	err := vs.StateGen.SaveState(ctx,bytesutil.ToBytes32(headBlock.BlockRoot),beaconState)  //bytesutil.ToBytes32(
	require.NoError(t,err)


	pKT := make([]*ethpb.PubKeyTarget, 0)
	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: privKeys[0].PublicKey().Marshal(),
		TargetEpoch: types.Epoch(types.Slot(80)/ params.BeaconConfig().SlotsPerEpoch)})
	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	_, err = vs.DetectDoppelganger(ctx, req)
	require.NoError(t, err)
	// This section was not working.
	/*service :=  stategen.New(beaconDB)


	prevState, newSignedblock, privKeys, _, _, _ := createFullBlockWithOperations(t)
	newState, err := state.ProcessBlock(context.Background(), prevState, interfaces.WrappedPhase0SignedBeaconBlock(newSignedblock))
	require.NoError(t, err)
	require.NoError(t, newState.SetSlot(200))
	newRoot,err := newSignedblock.Block.HashTreeRoot()
	require.NoError(t, err)

	vs := &Server{
		BeaconDB:           beaconDB,
		Ctx:                ctx,
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		HeadFetcher:        &mockChain.ChainService{State: newState, Root: newRoot[:]},
		StateGen:           service,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &pbp2p.StateSummary{Root: newRoot[:]}))
	err = vs.StateGen.SaveState(ctx,newRoot,newState)  //bytesutil.ToBytes32(
	require.NoError(t,err)

	pState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, pState.SetSlot(420))
	blk := testutil.NewBeaconBlock()
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &pbp2p.StateSummary{Root: blkRoot[:]}))
	pKT := make([]*ethpb.PubKeyTarget, 0)
	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: privKeys[0].PublicKey().Marshal(),
		TargetEpoch: types.Epoch(types.Slot(400)/ params.BeaconConfig().SlotsPerEpoch)})
	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	_, err = vs.DetectDoppelganger(ctx, req)
	require.NoError(t, err)+



} */

func Test_DetectDoppelganger_NoPrevState3(t *testing.T){

	prevConfig := params.BeaconConfig().Copy()
	defer params.OverrideBeaconConfig(prevConfig)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	cfg := &blockchain.Config{
		BeaconDB:        beaconDB,
		StateGen:        stategen.New(beaconDB),
		ForkChoiceStore: protoarray.New(0, 0, [32]byte{}),
		DepositCache:    depositCache,
		StateNotifier:   &mockBC.MockStateNotifier{},
	}
	service, err := blockchain.NewService(ctx, cfg)
	require.NoError(t, err)

	gs, keys := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, service.SaveGenesisData(ctx, gs))
	err = cfg.BeaconDB.SaveGenesisData(ctx, gs)
	require.NoError(t,err)
	genesisBlk, err := cfg.BeaconDB.GenesisBlock(ctx)
	require.NoError(t,err)
	genesisBlkRoot, err := genesisBlk.Block().HashTreeRoot()
	require.NoError(t,err)
	cfg.StateGen.SaveFinalizedState(0 /*slot*/, genesisBlkRoot, gs)
	require.NoError(t,service.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: genesisBlkRoot[:]}))

	testState := gs.Copy()
	for i := types.Slot(1); i <= 4*params.BeaconConfig().SlotsPerEpoch; i++ {
		blk, err := testutil.GenerateFullBlock(testState, keys, testutil.DefaultBlockGenConfig(), i)
		require.NoError(t, err)
		r, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, service.OnBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(blk), r))
		_, err =state.ExecuteStateTransition(ctx,testState,interfaces.WrappedPhase0SignedBeaconBlock(blk))
		require.NoError(t, err)
		testState, err = cfg.StateGen.StateByRoot(ctx, r)
		require.NoError(t, err)
	}
	require.Equal(t, types.Epoch(3), service.CurrentJustifiedCheckpt().Epoch)
	require.Equal(t, types.Epoch(2), service.FinalizedCheckpt().Epoch)

	tRoot,err := testState.HashTreeRoot(ctx)
	require.NoError(t, err)

	vs := &Server{
		BeaconDB:           beaconDB,
		Ctx:                ctx,
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		HeadFetcher:        &mockChain.ChainService{State: testState, Root: tRoot[:]},
		StateGen:           cfg.StateGen,
	}
	pKT := make([]*ethpb.PubKeyTarget, 0)
	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: keys[0].PublicKey().Marshal(),
		TargetEpoch: types.Epoch(types.Slot(40) / params.BeaconConfig().SlotsPerEpoch)})
	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	_, err = vs.DetectDoppelganger(ctx, req)
	require.NoError(t,err)
	// No Previous state is available
	//assert.ErrorContains(t, "Doppelganger rpc service - Could not get previous state root", err)
}

func fillDBTestFullBlocksWithOperations(ctx context.Context, t *testing.T, beaconDB db.Database) (iface.BeaconState,
	*ethpb.SignedBeaconBlock, []bls.SecretKey, []*ethpb.BeaconBlockContainer) {

	parentRoot := [32]byte{1, 2, 3}
	genBlk := testutil.NewBeaconBlock()
	genBlk.Block.ParentRoot = parentRoot[:]
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(genBlk)))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]interfaces.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb.BeaconBlockContainer, count)

	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)

	for i := types.Slot(0); i < count; i++ {
		newState, b, _, _, _, _ := createFullBlockFromDB(t, beaconState, privKeys,i,ctx)
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte{uint8(i)}, 32)
		att1 := testutil.NewAttestation()
		att1.Data.Slot = i
		att1.Data.CommitteeIndex = types.CommitteeIndex(i)
		att2 := testutil.NewAttestation()
		att2.Data.Slot = i
		att2.Data.CommitteeIndex = types.CommitteeIndex(i + 1)
		b.Block.Body.Attestations = []*ethpb.Attestation{att1, att2}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i] = interfaces.WrappedPhase0SignedBeaconBlock(b)
		blkContainers[i] = &ethpb.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
		beaconState, err = state.ProcessBlock(context.Background(), newState, interfaces.WrappedPhase0SignedBeaconBlock(b))
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(b)))

		//beaconState = newState
	}

	return beaconState, genBlk, privKeys, blkContainers
}

//  my version of createFullBlockWithOperations just passed it the state and privKeys
func createFullBlockFromDB(t *testing.T, beaconState iface.BeaconState, privKeys []bls.SecretKey, slot types.Slot,ctx context.Context) (iface.BeaconState,
	*ethpb.SignedBeaconBlock, []bls.SecretKey, []*ethpb.Attestation, []*ethpb.ProposerSlashing, []*ethpb.SignedVoluntaryExit) {
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := genesisBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	err = beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	require.NoError(t, err)
	err = beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector))
	require.NoError(t, err)
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cp))
	require.NoError(t, beaconState.AppendCurrentEpochAttestations(&pbp2p.PendingAttestation{}))

	proposerSlashIdx := types.ValidatorIndex(3)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	err = beaconState.SetSlot(slotsPerEpoch.Mul(uint64(params.BeaconConfig().ShardCommitteePeriod)) + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

	currentEpoch := helpers.CurrentEpoch(beaconState)
	header1 := testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerSlashIdx,
			Slot:          1,
			StateRoot:     bytesutil.PadTo([]byte("A"), 32),
		},
	})
	header1.Signature, err = helpers.ComputeDomainAndSign(beaconState, currentEpoch, header1.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerSlashIdx])
	require.NoError(t, err)

	header2 := testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerSlashIdx,
			Slot:          1,
			StateRoot:     bytesutil.PadTo([]byte("B"), 32),
		},
	})
	header2.Signature, err = helpers.ComputeDomainAndSign(beaconState, helpers.CurrentEpoch(beaconState), header2.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerSlashIdx])
	require.NoError(t, err)

	proposerSlashings := []*ethpb.ProposerSlashing{
		{
			Header_1: header1,
			Header_2: header2,
		},
	}
	validators := beaconState.Validators()
	validators[proposerSlashIdx].PublicKey = privKeys[proposerSlashIdx].PublicKey().Marshal()
	require.NoError(t, beaconState.SetValidators(validators))

	mockRoot2 := [32]byte{'A'}
	att1 := testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot2[:]},
		},
		AttestingIndices: []uint64{0, 1},
	})
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	hashTreeRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	require.NoError(t, err)
	sig0 := privKeys[0].Sign(hashTreeRoot[:])
	sig1 := privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig := bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()

	mockRoot3 := [32]byte{'B'}
	att2 := testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot3[:]},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		AttestingIndices: []uint64{0, 1},
	})

	hashTreeRoot, err = helpers.ComputeSigningRoot(att2.Data, domain)
	require.NoError(t, err)
	sig0 = privKeys[0].Sign(hashTreeRoot[:])
	sig1 = privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig = bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()

	attesterSlashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}

	var blockRoots [][]byte
	for i := uint64(0); i < uint64(params.BeaconConfig().SlotsPerHistoricalRoot); i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	require.NoError(t, beaconState.SetBlockRoots(blockRoots))

	aggBits := bitfield.NewBitlist(1)
	aggBits.SetBitAt(0, true)
	blockAtt := testutil.HydrateAttestation(&ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:   beaconState.Slot(),
			Target: &ethpb.Checkpoint{Epoch: helpers.CurrentEpoch(beaconState)},
			Source: &ethpb.Checkpoint{Root: mockRoot[:]}},
		AggregationBits: aggBits,
	})

	committee, err := helpers.BeaconCommitteeFromState(beaconState, blockAtt.Data.Slot, blockAtt.Data.CommitteeIndex)
	assert.NoError(t, err)
	attestingIndices, err := attestationutil.AttestingIndices(blockAtt.AggregationBits, committee)
	require.NoError(t, err)
	assert.NoError(t, err)
	hashTreeRoot, err = helpers.ComputeSigningRoot(blockAtt.Data, domain)
	assert.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	blockAtt.Signature = bls.AggregateSignatures(sigs).Marshal()

	exit := &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			ValidatorIndex: 10,
			Epoch:          params.BeaconConfig().FarFutureEpoch,
		},
	}
	exit.Signature, err = helpers.ComputeDomainAndSign(beaconState, currentEpoch, exit.Exit, params.BeaconConfig().DomainVoluntaryExit, privKeys[exit.Exit.ValidatorIndex])
	require.NoError(t, err)

	header := beaconState.LatestBlockHeader()
	prevStateRoot, err := beaconState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	header.StateRoot = prevStateRoot[:]
	require.NoError(t, beaconState.SetLatestBlockHeader(header))
	parentRoot, err := beaconState.LatestBlockHeader().HashTreeRoot()
	require.NoError(t, err)
	copied := beaconState.Copy()
	require.NoError(t, copied.SetSlot(beaconState.Slot()+1))
	randaoReveal, err := testutil.RandaoReveal(copied, currentEpoch, privKeys)
	require.NoError(t, err)
	proposerIndex, err := helpers.BeaconProposerIndex(copied)
	require.NoError(t, err)
	block := testutil.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot:    parentRoot[:],
			Slot:          beaconState.Slot() + 1,
			ProposerIndex: proposerIndex,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal:      randaoReveal,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      []*ethpb.Attestation{blockAtt},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{exit},
			},
		},
	})

	sig, err := testutil.BlockSignature(beaconState, block.Block, privKeys)
	require.NoError(t, err)
	block.Signature = sig.Marshal()

	require.NoError(t, beaconState.SetSlot(block.Block.Slot))
	return beaconState, block, privKeys, []*ethpb.Attestation{blockAtt}, proposerSlashings, []*ethpb.SignedVoluntaryExit{exit}
}

// COPIED FUNC from elsewhere
//  used func from block_test.go
func fillDBTestBlocks(ctx context.Context, t *testing.T, beaconDB db.Database) (*ethpb.SignedBeaconBlock, []*ethpb.BeaconBlockContainer) {
	parentRoot := [32]byte{1, 2, 3}
	genBlk := testutil.NewBeaconBlock()
	genBlk.Block.ParentRoot = parentRoot[:]
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, interfaces.WrappedPhase0SignedBeaconBlock(genBlk)))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]interfaces.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb.BeaconBlockContainer, count)
	for i := types.Slot(0); i < count; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte{uint8(i)}, 32)
		att1 := testutil.NewAttestation()
		att1.Data.Slot = i
		att1.Data.CommitteeIndex = types.CommitteeIndex(i)
		att2 := testutil.NewAttestation()
		att2.Data.Slot = i
		att2.Data.CommitteeIndex = types.CommitteeIndex(i + 1)
		b.Block.Body.Attestations = []*ethpb.Attestation{att1, att2}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i] = interfaces.WrappedPhase0SignedBeaconBlock(b)
		blkContainers[i] = &ethpb.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
	}
	require.NoError(t, beaconDB.SaveBlocks(ctx, blks))
	headRoot := bytesutil.ToBytes32(blkContainers[len(blks)-1].BlockRoot)
	summary := &pbp2p.StateSummary{
		Root: headRoot[:],
		Slot: blkContainers[len(blks)-1].Block.Block.Slot,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, summary))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, headRoot))
	return genBlk, blkContainers
}

//  used func from transition_test.go, returned the privKeys
func createFullBlockWithOperations(t *testing.T) (iface.BeaconState,
	*ethpb.SignedBeaconBlock, []bls.SecretKey, []*ethpb.Attestation, []*ethpb.ProposerSlashing, []*ethpb.SignedVoluntaryExit) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := genesisBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	err = beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	require.NoError(t, err)
	err = beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector))
	require.NoError(t, err)
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cp))
	require.NoError(t, beaconState.AppendCurrentEpochAttestations(&pbp2p.PendingAttestation{}))

	proposerSlashIdx := types.ValidatorIndex(3)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	err = beaconState.SetSlot(slotsPerEpoch.Mul(uint64(params.BeaconConfig().ShardCommitteePeriod)) + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

	currentEpoch := helpers.CurrentEpoch(beaconState)
	header1 := testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerSlashIdx,
			Slot:          1,
			StateRoot:     bytesutil.PadTo([]byte("A"), 32),
		},
	})
	header1.Signature, err = helpers.ComputeDomainAndSign(beaconState, currentEpoch, header1.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerSlashIdx])
	require.NoError(t, err)

	header2 := testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerSlashIdx,
			Slot:          1,
			StateRoot:     bytesutil.PadTo([]byte("B"), 32),
		},
	})
	header2.Signature, err = helpers.ComputeDomainAndSign(beaconState, helpers.CurrentEpoch(beaconState), header2.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerSlashIdx])
	require.NoError(t, err)

	proposerSlashings := []*ethpb.ProposerSlashing{
		{
			Header_1: header1,
			Header_2: header2,
		},
	}
	validators := beaconState.Validators()
	validators[proposerSlashIdx].PublicKey = privKeys[proposerSlashIdx].PublicKey().Marshal()
	require.NoError(t, beaconState.SetValidators(validators))

	mockRoot2 := [32]byte{'A'}
	att1 := testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot2[:]},
		},
		AttestingIndices: []uint64{0, 1},
	})
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	hashTreeRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	require.NoError(t, err)
	sig0 := privKeys[0].Sign(hashTreeRoot[:])
	sig1 := privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig := bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()

	mockRoot3 := [32]byte{'B'}
	att2 := testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot3[:]},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		AttestingIndices: []uint64{0, 1},
	})

	hashTreeRoot, err = helpers.ComputeSigningRoot(att2.Data, domain)
	require.NoError(t, err)
	sig0 = privKeys[0].Sign(hashTreeRoot[:])
	sig1 = privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig = bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()

	attesterSlashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}

	var blockRoots [][]byte
	for i := uint64(0); i < uint64(params.BeaconConfig().SlotsPerHistoricalRoot); i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	require.NoError(t, beaconState.SetBlockRoots(blockRoots))

	aggBits := bitfield.NewBitlist(1)
	aggBits.SetBitAt(0, true)
	blockAtt := testutil.HydrateAttestation(&ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:   beaconState.Slot(),
			Target: &ethpb.Checkpoint{Epoch: helpers.CurrentEpoch(beaconState)},
			Source: &ethpb.Checkpoint{Root: mockRoot[:]}},
		AggregationBits: aggBits,
	})

	committee, err := helpers.BeaconCommitteeFromState(beaconState, blockAtt.Data.Slot, blockAtt.Data.CommitteeIndex)
	assert.NoError(t, err)
	attestingIndices, err := attestationutil.AttestingIndices(blockAtt.AggregationBits, committee)
	require.NoError(t, err)
	assert.NoError(t, err)
	hashTreeRoot, err = helpers.ComputeSigningRoot(blockAtt.Data, domain)
	assert.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	blockAtt.Signature = bls.AggregateSignatures(sigs).Marshal()

	exit := &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			ValidatorIndex: 10,
			Epoch:          0,
		},
	}
	exit.Signature, err = helpers.ComputeDomainAndSign(beaconState, currentEpoch, exit.Exit, params.BeaconConfig().DomainVoluntaryExit, privKeys[exit.Exit.ValidatorIndex])
	require.NoError(t, err)

	header := beaconState.LatestBlockHeader()
	prevStateRoot, err := beaconState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	header.StateRoot = prevStateRoot[:]
	require.NoError(t, beaconState.SetLatestBlockHeader(header))
	parentRoot, err := beaconState.LatestBlockHeader().HashTreeRoot()
	require.NoError(t, err)
	copied := beaconState.Copy()
	require.NoError(t, copied.SetSlot(beaconState.Slot()+1))
	randaoReveal, err := testutil.RandaoReveal(copied, currentEpoch, privKeys)
	require.NoError(t, err)
	proposerIndex, err := helpers.BeaconProposerIndex(copied)
	require.NoError(t, err)
	block := testutil.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot:    parentRoot[:],
			Slot:          beaconState.Slot() + 1,
			ProposerIndex: proposerIndex,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal:      randaoReveal,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      []*ethpb.Attestation{blockAtt},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{exit},
			},
		},
	})

	sig, err := testutil.BlockSignature(beaconState, block.Block, privKeys)
	require.NoError(t, err)
	block.Signature = sig.Marshal()

	require.NoError(t, beaconState.SetSlot(block.Block.Slot))
	return beaconState, block, privKeys, []*ethpb.Attestation{blockAtt}, proposerSlashings, []*ethpb.SignedVoluntaryExit{exit}
}

// blockTree1 constructs the following tree:
//    /- B1
// B0           /- B5 - B7
//    \- B3 - B4 - B6 - B8
// (B1, and B3 are all from the same slots)
func blockTree1(t *testing.T, beaconDB db.Database, genesisRoot []byte) ([][]byte, error,[]bls.SecretKey,iface.BeaconState ) {
	genesisRoot = bytesutil.PadTo(genesisRoot, 32)
	b0 := testutil.NewBeaconBlock()
	b0.Block.Slot = 100
	b0.Block.ParentRoot = genesisRoot
	r0, err := b0.Block.HashTreeRoot()
	if err != nil {
		return nil, err,nil,nil
	}
	b1 := testutil.NewBeaconBlock()
	b1.Block.Slot = 101
	b1.Block.ParentRoot = r0[:]
	r1, err := b1.Block.HashTreeRoot()
	if err != nil {
		return nil, err,nil,nil
	}
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 103
	b3.Block.ParentRoot = r0[:]
	r3, err := b3.Block.HashTreeRoot()
	if err != nil {
		return nil, err,nil,nil
	}
	b4 := testutil.NewBeaconBlock()
	b4.Block.Slot = 104
	b4.Block.ParentRoot = r3[:]
	r4, err := b4.Block.HashTreeRoot()
	if err != nil {
		return nil, err,nil,nil
	}
	b5 := testutil.NewBeaconBlock()
	b5.Block.Slot = 105
	b5.Block.ParentRoot = r4[:]
	r5, err := b5.Block.HashTreeRoot()
	if err != nil {
		return nil, err,nil,nil
	}
	b6 := testutil.NewBeaconBlock()
	b6.Block.Slot = 106
	b6.Block.ParentRoot = r4[:]
	r6, err := b6.Block.HashTreeRoot()
	if err != nil {
		return nil, err,nil,nil
	}
	b7 := testutil.NewBeaconBlock()
	b7.Block.Slot = 107
	b7.Block.ParentRoot = r5[:]
	r7, err := b7.Block.HashTreeRoot()
	if err != nil {
		return nil, err,nil,nil
	}
	b8 := testutil.NewBeaconBlock()
	b8.Block.Slot = 108
	b8.Block.ParentRoot = r6[:]
	r8, err := b8.Block.HashTreeRoot()
	if err != nil {
		return nil, err,nil,nil
	}
	//st, err := testutil.NewBeaconState()
	st, privKeys := testutil.DeterministicGenesisState(t, 32)
	//require.NoError(t, err)

	for _, b := range []*ethpb.SignedBeaconBlock{b0, b1, b3, b4, b5, b6, b7, b8} {
		beaconBlock := testutil.NewBeaconBlock()
		beaconBlock.Block.Slot = b.Block.Slot
		beaconBlock.Block.ParentRoot = bytesutil.PadTo(b.Block.ParentRoot, 32)
		if err := beaconDB.SaveBlock(context.Background(), interfaces.WrappedPhase0SignedBeaconBlock(beaconBlock)); err != nil {
			return nil, err,nil,nil
		}
		if err := beaconDB.SaveState(context.Background(), st.Copy(), bytesutil.ToBytes32(beaconBlock.Block.ParentRoot)); err != nil {
			return nil, errors.Wrap(err, "could not save state"),nil,nil
		}
	}
	if err := beaconDB.SaveState(context.Background(), st.Copy(), r1); err != nil {
		return nil, err,nil,nil
	}
	if err := beaconDB.SaveState(context.Background(), st.Copy(), r7); err != nil {
		return nil, err,nil,nil
	}
	if err := beaconDB.SaveState(context.Background(), st.Copy(), r8); err != nil {
		return nil, err,nil,nil
	}
	return [][]byte{r0[:], r1[:], nil, r3[:], r4[:], r5[:], r6[:], r7[:], r8[:]}, nil,privKeys,st
}