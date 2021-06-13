package validator

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
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

func Test_DetectDoppelganger_TargetHeadClose(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	priv1, err := bls.RandKey()
	require.NoError(t, err)

	pubKey1 := priv1.PublicKey().Marshal()
	slot := types.Slot(4000)
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
	prevState, newblock, _, _, _ := createFullBlockWithOperations(t)
	newState, err := state.ProcessBlock(context.Background(), prevState, interfaces.WrappedPhase0SignedBeaconBlock(newblock))
	require.NoError(t, err)
	newRoot, err := newblock.Block.HashTreeRoot()
	require.NoError(t, err)
	vs.HeadFetcher = &mockChain.ChainService{State: newState, Root: newRoot[:]}

	//service := stategen.New(beaconDB)
	//err = service.SaveState(ctx,newRoot,newState)
	//require.NoError(t, err)
	//service.hotStateCache.put(bytesutil.ToBytes32(b1.Block.ParentRoot), genesis)
	_, err = vs.DetectDoppelganger(ctx, req)
	require.NoError(t, err)
}

func createFullBlockWithOperations(t *testing.T) (iface.BeaconState,
	*ethpb.SignedBeaconBlock, []*ethpb.Attestation, []*ethpb.ProposerSlashing, []*ethpb.SignedVoluntaryExit) {
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

	aggBits := bitfield.NewBitlist(4)
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
	return beaconState, block, []*ethpb.Attestation{blockAtt}, proposerSlashings, []*ethpb.SignedVoluntaryExit{exit}
}

/*
func Test_DetectDoppelganger_NoPrevState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	priv1, err := bls.RandKey()
	require.NoError(t, err)

	pubKey1 := priv1.PublicKey().Marshal()
	slot := types.Slot(4000)
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
	}

	pKT := make([]*ethpb.PubKeyTarget, 0)

	pKT = append(pKT, &ethpb.PubKeyTarget{PubKey: pubKey1,
		TargetEpoch: types.Epoch(slot.Sub(20) / params.BeaconConfig().SlotsPerEpoch)})
	req := &ethpb.DetectDoppelgangerRequest{
		PubKeysTargets: pKT}
	_, err = vs.DetectDoppelganger(ctx, req)
	// No Previous state is available
	assert.ErrorContains(t, "Doppelganger rpc service - Could not get previous state root", err)

}

func Test_DetectDoppelganger_BalanceCalculate(t *testing.T){

}

logrus.SetLevel(logrus.PanicLevel)

	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount * 4
	committeeCount := validatorCount / params.BeaconConfig().TargetCommitteeSize
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:          params.BeaconConfig().FarFutureEpoch,
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		randaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}

	base := &pb.BeaconState{
		Slot:              20,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{},
		BlockRoots:        make([][]byte, 254),
		RandaoMixes:       randaoMixes,
		Validators:        validators,
		Balances:          validatorBalances,
		Slashings:         make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Root: []byte("hello-world"),
		},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
	}
	s, err := stateV0.InitializeFromProto(base)
	require.NoError(b, err)


*/
/*
func TestProcessBlk_AttsBasedOnValidatorCount(t *testing.T) {
	logrus.SetLevel(logrus.PanicLevel)

	// Default at 256 validators, can raise this number with faster BLS.
	validatorCount := uint64(256)
	s, privKeys := testutil.DeterministicGenesisState(t, validatorCount)
	require.NoError(t, s.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	bitCount := validatorCount / uint64(params.BeaconConfig().SlotsPerEpoch)
	aggBits := bitfield.NewBitlist(bitCount)
	for i := uint64(1); i < bitCount; i++ {
		aggBits.SetBitAt(i, true)
	}
	atts := make([]*ethpb.Attestation, 1)

	for i := 0; i < len(atts); i++ {
		att := testutil.HydrateAttestation(&ethpb.Attestation{
			Data:            &ethpb.AttestationData{Slot: 1},
			AggregationBits: aggBits,
		})

		committee, err := helpers.BeaconCommitteeFromState(s, att.Data.Slot, att.Data.CommitteeIndex)
		assert.NoError(t, err)
		attestingIndices, err := attestationutil.AttestingIndices(att.AggregationBits, committee)
		require.NoError(t, err)
		domain, err := helpers.Domain(s.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, s.GenesisValidatorRoot())
		require.NoError(t, err)
		sigs := make([]bls.Signature, len(attestingIndices))
		for i, indice := range attestingIndices {
			hashTreeRoot, err := helpers.ComputeSigningRoot(att.Data, domain)
			assert.NoError(t, err)
			sig := privKeys[indice].Sign(hashTreeRoot[:])
			sigs[i] = sig
		}
		att.Signature = bls.AggregateSignatures(sigs).Marshal()
		atts[i] = att
	}

	copied := s.Copy()
	require.NoError(t, copied.SetSlot(s.Slot()+1))
	epochSignature, err := testutil.RandaoReveal(copied, helpers.CurrentEpoch(copied), privKeys)
	require.NoError(t, err)
	header := s.LatestBlockHeader()
	prevStateRoot, err := s.HashTreeRoot(context.Background())
	require.NoError(t, err)
	header.StateRoot = prevStateRoot[:]
	require.NoError(t, s.SetLatestBlockHeader(header))

	parentRoot, err := s.LatestBlockHeader().HashTreeRoot()
	require.NoError(t, err)

	nextSlotState := s.Copy()
	require.NoError(t, nextSlotState.SetSlot(s.Slot()+1))
	proposerIdx, err := helpers.BeaconProposerIndex(nextSlotState)
	require.NoError(t, err)
	blk := testutil.NewBeaconBlock()
	blk.Block.ProposerIndex = proposerIdx
	blk.Block.Slot = s.Slot() + 1
	blk.Block.ParentRoot = parentRoot[:]
	blk.Block.Body.RandaoReveal = epochSignature
	blk.Block.Body.Attestations = atts
	sig, err := testutil.BlockSignature(s, blk.Block, privKeys)
	require.NoError(t, err)
	blk.Signature = sig.Marshal()

	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.MinAttestationInclusionDelay = 0
	params.OverrideBeaconConfig(config)

	require.NoError(t, s.SetSlot(s.Slot()+1))
	_, err = state.ProcessBlock(context.Background(), s, interfaces.WrappedPhase0SignedBeaconBlock(blk))
	require.NoError(t, err)
}

*/
