package validator

import (
	"context"
	"math/big"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation"
	attaggregation "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/protobuf/proto"
)

func TestProposer_GetBlock_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := util.DeterministicGenesisState(t, 64)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := &Server{
		HeadFetcher:       &mock.ChainService{State: beaconState, Root: parentRoot[:]},
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		BlockReceiver:     &mock.ChainService{},
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		MockEth1Votes:     true,
		AttPool:           attestations.NewPool(),
		SlashingsPool:     slashings.NewPool(),
		ExitPool:          voluntaryexits.NewPool(),
		StateGen:          stategen.New(db),
	}

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	req := &ethpb.BlockRequest{
		Slot:         1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}

	proposerSlashings := make([]*ethpb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := types.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
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
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
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

	assert.Equal(t, req.Slot, block.Slot, "Expected block to have slot of 1")
	assert.DeepEqual(t, parentRoot[:], block.ParentRoot, "Expected block to have correct parent root")
	assert.DeepEqual(t, randaoReveal, block.Body.RandaoReveal, "Expected block to have correct randao reveal")
	assert.DeepEqual(t, req.Graffiti, block.Body.Graffiti, "Expected block to have correct graffiti")
	assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(block.Body.ProposerSlashings)))
	assert.DeepEqual(t, proposerSlashings, block.Body.ProposerSlashings)
	assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(block.Body.AttesterSlashings)))
	assert.DeepEqual(t, attSlashings, block.Body.AttesterSlashings)
}

func TestProposer_GetBlock_AddsUnaggregatedAtts(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := util.DeterministicGenesisState(t, params.BeaconConfig().MinGenesisActiveValidatorCount)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := &Server{
		HeadFetcher:       &mock.ChainService{State: beaconState, Root: parentRoot[:]},
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		BlockReceiver:     &mock.ChainService{},
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		MockEth1Votes:     true,
		SlashingsPool:     slashings.NewPool(),
		AttPool:           attestations.NewPool(),
		ExitPool:          voluntaryexits.NewPool(),
		StateGen:          stategen.New(db),
	}

	// Generate a bunch of random attestations at slot. These would be considered double votes, but
	// we don't care for the purpose of this test.
	var atts []*ethpb.Attestation
	for i := uint64(0); len(atts) < int(params.BeaconConfig().MaxAttestations); i++ {
		a, err := util.GenerateAttestations(beaconState, privKeys, 4, 1, true)
		require.NoError(t, err)
		atts = append(atts, a...)
	}
	// Max attestations minus one so we can almost fill the block and then include 1 unaggregated
	// att to maximize inclusion.
	atts = atts[:params.BeaconConfig().MaxAttestations-1]
	require.NoError(t, proposerServer.AttPool.SaveAggregatedAttestations(atts))

	// Generate some more random attestations with a larger spread so that we can capture at least
	// one unaggregated attestation.
	atts, err = util.GenerateAttestations(beaconState, privKeys, 300, 1, true)
	require.NoError(t, err)
	found := false
	for _, a := range atts {
		if !helpers.IsAggregated(a) {
			found = true
			require.NoError(t, proposerServer.AttPool.SaveUnaggregatedAttestation(a))
		}
	}
	require.Equal(t, true, found, "No unaggregated attestations were generated")

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	assert.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	req := &ethpb.BlockRequest{
		Slot:         1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}
	block, err := proposerServer.GetBlock(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, req.Slot, block.Slot, "Expected block to have slot of 1")
	assert.DeepEqual(t, parentRoot[:], block.ParentRoot, "Expected block to have correct parent root")
	assert.DeepEqual(t, randaoReveal, block.Body.RandaoReveal, "Expected block to have correct randao reveal")
	assert.DeepEqual(t, req.Graffiti, block.Body.Graffiti, "Expected block to have correct graffiti")
	assert.Equal(t, params.BeaconConfig().MaxAttestations, uint64(len(block.Body.Attestations)), "Expected block atts to be aggregated down to 1")
	hasUnaggregatedAtt := false
	for _, a := range block.Body.Attestations {
		if !helpers.IsAggregated(a) {
			hasUnaggregatedAtt = true
			break
		}
	}
	assert.Equal(t, false, hasUnaggregatedAtt, "Expected block to not have unaggregated attestation")
}

func TestProposer_ProposeBlock_Phase0_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	genesis := util.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

	numDeposits := uint64(64)
	beaconState, _ := util.DeterministicGenesisState(t, numDeposits)
	bsRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

	c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
	proposerServer := &Server{
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		BlockReceiver:     c,
		HeadFetcher:       c,
		BlockNotifier:     c.BlockNotifier(),
		P2P:               mockp2p.NewTestP2P(t),
	}
	req := util.NewBeaconBlock()
	req.Block.Slot = 5
	req.Block.ParentRoot = bsRoot[:]
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(req)))
	blk := &ethpb.GenericSignedBeaconBlock_Phase0{Phase0: req}
	_, err = proposerServer.ProposeBeaconBlock(context.Background(), &ethpb.GenericSignedBeaconBlock{Block: blk})
	assert.NoError(t, err, "Could not propose block correctly")
}

func TestProposer_ProposeBlock_Altair_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	genesis := util.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

	numDeposits := uint64(64)
	beaconState, _ := util.DeterministicGenesisStateAltair(t, numDeposits)
	bsRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

	c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
	proposerServer := &Server{
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		BlockReceiver:     c,
		HeadFetcher:       c,
		BlockNotifier:     c.BlockNotifier(),
		P2P:               mockp2p.NewTestP2P(t),
	}
	blockToPropose := util.NewBeaconBlockAltair()
	blockToPropose.Block.Slot = 5
	blockToPropose.Block.ParentRoot = bsRoot[:]
	blk := &ethpb.GenericSignedBeaconBlock_Altair{Altair: blockToPropose}
	wrapped, err := wrapper.WrappedAltairSignedBeaconBlock(blockToPropose)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wrapped))
	_, err = proposerServer.ProposeBeaconBlock(context.Background(), &ethpb.GenericSignedBeaconBlock{Block: blk})
	assert.NoError(t, err, "Could not propose block correctly")
}

func TestProposer_ComputeStateRoot_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := util.DeterministicGenesisState(t, 100)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := &Server{
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		StateGen:          stategen.New(db),
	}
	req := util.NewBeaconBlock()
	req.Block.ProposerIndex = 21
	req.Block.ParentRoot = parentRoot[:]
	req.Block.Slot = 1
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+1))
	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)
	proposerIdx, err := helpers.BeaconProposerIndex(ctx, beaconState)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()-1))
	req.Block.Body.RandaoReveal = randaoReveal
	currentEpoch := time.CurrentEpoch(beaconState)
	req.Signature, err = signing.ComputeDomainAndSign(beaconState, currentEpoch, req.Block, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerIdx])
	require.NoError(t, err)

	_, err = proposerServer.computeStateRoot(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(req))
	require.NoError(t, err)
}

func TestProposer_PendingDeposits_Eth1DataVoteOK(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	newHeight := big.NewInt(height.Int64() + 11000)
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()):    []byte("0x0"),
			int(newHeight.Int64()): []byte("0x1"),
		},
	}

	var votes []*ethpb.Eth1Data

	blockHash := make([]byte, 32)
	copy(blockHash, "0x1")
	vote := &ethpb.Eth1Data{
		DepositRoot:  make([]byte, 32),
		BlockHash:    blockHash,
		DepositCount: 3,
	}
	period := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod)))
	for i := 0; i <= int(period/2); i++ {
		votes = append(votes, vote)
	}

	blockHash = make([]byte, 32)
	copy(blockHash, "0x0")
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetEth1DepositIndex(2))
	require.NoError(t, beaconState.SetEth1Data(&ethpb.Eth1Data{
		DepositRoot:  make([]byte, 32),
		BlockHash:    blockHash,
		DepositCount: 2,
	}))
	require.NoError(t, beaconState.SetEth1DataVotes(votes))

	blk := util.NewBeaconBlock()
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	bs := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockReceiver:     &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:       &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	_, eth1Height, err := bs.canonicalEth1Data(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)

	assert.Equal(t, 0, eth1Height.Cmp(height))

	newState, err := b.ProcessEth1DataInBlock(ctx, beaconState, blk.Block.Body.Eth1Data)
	require.NoError(t, err)

	if proto.Equal(newState.Eth1Data(), vote) {
		t.Errorf("eth1data in the state equal to vote, when not expected to"+
			"have majority: Got %v", vote)
	}

	blk.Block.Body.Eth1Data = vote

	_, eth1Height, err = bs.canonicalEth1Data(ctx, beaconState, vote)
	require.NoError(t, err)
	assert.Equal(t, 0, eth1Height.Cmp(newHeight))

	newState, err = b.ProcessEth1DataInBlock(ctx, beaconState, blk.Block.Body.Eth1Data)
	require.NoError(t, err)

	if !proto.Equal(newState.Eth1Data(), vote) {
		t.Errorf("eth1data in the state not of the expected kind: Got %v but wanted %v", newState.Eth1Data(), vote)
	}
}

func TestProposer_PendingDeposits_OutsideEth1FollowWindow(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:   bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot: make([]byte, 32),
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*ethpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 2,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 8,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*ethpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 400,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("c"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 600,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("d"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.HashTreeRoot()))
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.HashTreeRoot())
	}

	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()

	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)

	bs := &Server{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	deposits, err := bs.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected list of deposits")

	// It should not return the recent deposits after their follow window.
	// as latest block number makes no difference in retrieval of deposits
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err = bs.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_PendingDeposits_FollowsCorrectEth1Block(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	newHeight := big.NewInt(height.Int64() + 11000)
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()):    []byte("0x0"),
			int(newHeight.Int64()): []byte("0x1"),
		},
	}

	var votes []*ethpb.Eth1Data

	vote := &ethpb.Eth1Data{
		BlockHash:    bytesutil.PadTo([]byte("0x1"), 32),
		DepositRoot:  make([]byte, 32),
		DepositCount: 7,
	}
	period := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod)))
	for i := 0; i <= int(period/2); i++ {
		votes = append(votes, vote)
	}

	beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositRoot:  make([]byte, 32),
			DepositCount: 5,
		},
		Eth1DepositIndex: 1,
		Eth1DataVotes:    votes,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()

	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*ethpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 8,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 14,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*ethpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 5000,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("c"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 6000,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("d"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.HashTreeRoot()))
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.HashTreeRoot())
	}

	bs := &Server{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	deposits, err := bs.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected list of deposits")

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	// we should get our pending deposits once this vote pushes the vote tally to include
	// the updated eth1 data.
	deposits, err = bs.deposits(ctx, beaconState, vote)
	require.NoError(t, err)
	assert.Equal(t, len(recentDeposits), len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_PendingDeposits_CantReturnBelowStateEth1DepositIndex(t *testing.T) {
	ctx := context.Background()
	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetEth1Data(&ethpb.Eth1Data{
		BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
		DepositRoot:  make([]byte, 32),
		DepositCount: 100,
	}))
	require.NoError(t, beaconState.SetEth1DepositIndex(10))
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()
	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*ethpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*ethpb.DepositContainer
	for i := int64(2); i < 16; i++ {
		recentDeposits = append(recentDeposits, &ethpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{byte(i)}, 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.HashTreeRoot()))
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.HashTreeRoot())
	}

	bs := &Server{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err := bs.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)

	expectedDeposits := 6
	assert.Equal(t, expectedDeposits, len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_PendingDeposits_CantReturnMoreThanMax(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot:  make([]byte, 32),
			DepositCount: 100,
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()
	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*ethpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*ethpb.DepositContainer
	for i := int64(2); i < 22; i++ {
		recentDeposits = append(recentDeposits, &ethpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{byte(i)}, 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, height.Uint64(), dp.Index, depositTrie.HashTreeRoot()))
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, height.Uint64(), dp.Index, depositTrie.HashTreeRoot())
	}

	bs := &Server{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err := bs.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().MaxDeposits, uint64(len(deposits)), "Received unexpected number of pending deposits")
}

func TestProposer_PendingDeposits_CantReturnMoreThanDepositCount(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot:  make([]byte, 32),
			DepositCount: 5,
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()
	blkRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*ethpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*ethpb.DepositContainer
	for i := int64(2); i < 22; i++ {
		recentDeposits = append(recentDeposits, &ethpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{byte(i)}, 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.HashTreeRoot()))
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.HashTreeRoot())
	}

	bs := &Server{
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err := bs.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 3, len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_DepositTrie_UtilizesCachedFinalizedDeposits(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot:  make([]byte, 32),
			DepositCount: 4,
		},
		Eth1DepositIndex: 1,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()

	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	finalizedDeposits := []*ethpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*ethpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("c"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("d"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(finalizedDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.HashTreeRoot()))
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.HashTreeRoot())
	}

	bs := &Server{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	trie, err := bs.depositTrie(ctx, &ethpb.Eth1Data{}, big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance)))
	require.NoError(t, err)

	actualRoot := trie.HashTreeRoot()
	expectedRoot := depositTrie.HashTreeRoot()
	assert.Equal(t, expectedRoot, actualRoot, "Incorrect deposit trie root")
}

func TestProposer_DepositTrie_RebuildTrie(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot:  make([]byte, 32),
			DepositCount: 4,
		},
		Eth1DepositIndex: 1,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()

	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	finalizedDeposits := []*ethpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*ethpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("c"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("d"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(finalizedDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.HashTreeRoot()))
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.HashTreeRoot())
	}
	d := depositCache.AllDepositContainers(ctx)
	origDeposit, ok := proto.Clone(d[0].Deposit).(*ethpb.Deposit)
	assert.Equal(t, true, ok)
	junkCreds := mockCreds
	copy(junkCreds[:1], []byte{'A'})
	// Mutate it since its a pointer
	d[0].Deposit.Data.WithdrawalCredentials = junkCreds[:]
	// Insert junk to corrupt trie.
	depositCache.InsertFinalizedDeposits(ctx, 2)

	// Add original back
	d[0].Deposit = origDeposit

	bs := &Server{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
	}

	trie, err := bs.depositTrie(ctx, &ethpb.Eth1Data{}, big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance)))
	require.NoError(t, err)

	expectedRoot := depositTrie.HashTreeRoot()
	actualRoot := trie.HashTreeRoot()
	assert.Equal(t, expectedRoot, actualRoot, "Incorrect deposit trie root")

}

func TestProposer_ValidateDepositTrie(t *testing.T) {
	tt := []struct {
		name            string
		eth1dataCreator func() *ethpb.Eth1Data
		trieCreator     func() *trie.SparseMerkleTrie
		success         bool
	}{
		{
			name: "invalid trie items",
			eth1dataCreator: func() *ethpb.Eth1Data {
				return &ethpb.Eth1Data{DepositRoot: []byte{}, DepositCount: 10, BlockHash: []byte{}}
			},
			trieCreator: func() *trie.SparseMerkleTrie {
				trie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				return trie
			},
			success: false,
		},
		{
			name: "invalid deposit root",
			eth1dataCreator: func() *ethpb.Eth1Data {
				trie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				assert.NoError(t, trie.Insert([]byte{'a'}, 0))
				assert.NoError(t, trie.Insert([]byte{'b'}, 1))
				assert.NoError(t, trie.Insert([]byte{'c'}, 2))
				return &ethpb.Eth1Data{DepositRoot: []byte{'B'}, DepositCount: 3, BlockHash: []byte{}}
			},
			trieCreator: func() *trie.SparseMerkleTrie {
				trie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				assert.NoError(t, trie.Insert([]byte{'a'}, 0))
				assert.NoError(t, trie.Insert([]byte{'b'}, 1))
				assert.NoError(t, trie.Insert([]byte{'c'}, 2))
				return trie
			},
			success: false,
		},
		{
			name: "valid deposit trie",
			eth1dataCreator: func() *ethpb.Eth1Data {
				trie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				assert.NoError(t, trie.Insert([]byte{'a'}, 0))
				assert.NoError(t, trie.Insert([]byte{'b'}, 1))
				assert.NoError(t, trie.Insert([]byte{'c'}, 2))
				rt := trie.HashTreeRoot()
				return &ethpb.Eth1Data{DepositRoot: rt[:], DepositCount: 3, BlockHash: []byte{}}
			},
			trieCreator: func() *trie.SparseMerkleTrie {
				trie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
				assert.NoError(t, err)
				assert.NoError(t, trie.Insert([]byte{'a'}, 0))
				assert.NoError(t, trie.Insert([]byte{'b'}, 1))
				assert.NoError(t, trie.Insert([]byte{'c'}, 2))
				return trie
			},
			success: true,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			valid, err := validateDepositTrie(test.trieCreator(), test.eth1dataCreator())
			assert.Equal(t, test.success, valid)
			if valid {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProposer_Eth1Data_MajorityVote(t *testing.T) {
	slot := types.Slot(64)
	earliestValidTime, latestValidTime := majorityVoteBoundaryTime(slot)

	dc := ethpb.DepositContainer{
		Index:           0,
		Eth1BlockHeight: 0,
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             bytesutil.PadTo([]byte("a"), 48),
				Signature:             make([]byte, 96),
				WithdrawalCredentials: make([]byte, 32),
			}},
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err)
	depositCache, err := depositcache.New()
	require.NoError(t, err)
	assert.NoError(t, depositCache.InsertDeposit(context.Background(), dc.Deposit, dc.Eth1BlockHeight, dc.Index, depositTrie.HashTreeRoot()))

	t.Run("choose highest count", func(t *testing.T) {
		t.Skip()
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("second"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count at earliest valid time - choose highest count", func(t *testing.T) {
		t.Skip()
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("earliest"), DepositCount: 1},
				{BlockHash: []byte("earliest"), DepositCount: 1},
				{BlockHash: []byte("second"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("earliest")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count at latest valid time - choose highest count", func(t *testing.T) {
		t.Skip()
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("latest"), DepositCount: 1},
				{BlockHash: []byte("latest"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("latest")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count before range - choose highest count within range", func(t *testing.T) {
		t.Skip()
		p := mockPOW.NewPOWChain().
			InsertBlock(49, earliestValidTime-1, []byte("before_range")).
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("before_range"), DepositCount: 1},
				{BlockHash: []byte("before_range"), DepositCount: 1},
				{BlockHash: []byte("first"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count after range - choose highest count within range", func(t *testing.T) {
		t.Skip()
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(100, latestValidTime, []byte("latest")).
			InsertBlock(101, latestValidTime+1, []byte("after_range"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("after_range"), DepositCount: 1},
				{BlockHash: []byte("after_range"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count on unknown block - choose known block with highest count", func(t *testing.T) {
		t.Skip()
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("unknown"), DepositCount: 1},
				{BlockHash: []byte("unknown"), DepositCount: 1},
				{BlockHash: []byte("first"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no blocks in range - choose current eth1data", func(t *testing.T) {
		p := mockPOW.NewPOWChain().
			InsertBlock(49, earliestValidTime-1, []byte("before_range")).
			InsertBlock(101, latestValidTime+1, []byte("after_range"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
		})
		require.NoError(t, err)

		currentEth1Data := &ethpb.Eth1Data{DepositCount: 1, BlockHash: []byte("current")}
		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: currentEth1Data},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("current")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no votes in range - choose most recent block", func(t *testing.T) {
		p := mockPOW.NewPOWChain().
			InsertBlock(49, earliestValidTime-1, []byte("before_range")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(101, latestValidTime+1, []byte("after_range"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("before_range"), DepositCount: 1},
				{BlockHash: []byte("after_range"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := make([]byte, 32)
		copy(expectedHash, "second")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no votes - choose more recent block", func(t *testing.T) {
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot:          slot,
			Eth1DataVotes: []*ethpb.Eth1Data{}})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := make([]byte, 32)
		copy(expectedHash, "latest")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no votes and more recent block has less deposits - choose current eth1data", func(t *testing.T) {
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
		})
		require.NoError(t, err)

		// Set the deposit count in current eth1data to exceed the latest most recent block's deposit count.
		currentEth1Data := &ethpb.Eth1Data{DepositCount: 2, BlockHash: []byte("current")}
		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: currentEth1Data},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("current")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("same count - choose more recent block", func(t *testing.T) {
		t.Skip()
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("first"), DepositCount: 1},
				{BlockHash: []byte("second"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("second")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("highest count on block with less deposits - choose another block", func(t *testing.T) {
		t.Skip()
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(51, earliestValidTime+1, []byte("first")).
			InsertBlock(52, earliestValidTime+2, []byte("second")).
			InsertBlock(100, latestValidTime, []byte("latest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("no_new_deposits"), DepositCount: 0},
				{BlockHash: []byte("no_new_deposits"), DepositCount: 0},
				{BlockHash: []byte("second"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("second")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("only one block at earliest valid time - choose this block", func(t *testing.T) {
		t.Skip()
		p := mockPOW.NewPOWChain().InsertBlock(50, earliestValidTime, []byte("earliest"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("earliest"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("earliest")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("vote on last block before range - choose next block", func(t *testing.T) {
		p := mockPOW.NewPOWChain().
			InsertBlock(49, earliestValidTime-1, []byte("before_range")).
			// It is important to have height `50` with time `earliestValidTime+1` and not `earliestValidTime`
			// because of earliest block increment in the algorithm.
			InsertBlock(50, earliestValidTime+1, []byte("first"))

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("before_range"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 1}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := make([]byte, 32)
		copy(expectedHash, "first")
		assert.DeepEqual(t, expectedHash, hash)
	})

	t.Run("no deposits - choose chain start eth1data", func(t *testing.T) {
		p := mockPOW.NewPOWChain().
			InsertBlock(50, earliestValidTime, []byte("earliest")).
			InsertBlock(100, latestValidTime, []byte("latest"))
		p.Eth1Data = &ethpb.Eth1Data{
			BlockHash: []byte("eth1data"),
		}

		depositCache, err := depositcache.New()
		require.NoError(t, err)

		beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
			Slot: slot,
			Eth1DataVotes: []*ethpb.Eth1Data{
				{BlockHash: []byte("earliest"), DepositCount: 1},
			},
		})
		require.NoError(t, err)

		ps := &Server{
			ChainStartFetcher: p,
			Eth1InfoFetcher:   p,
			Eth1BlockFetcher:  p,
			BlockFetcher:      p,
			DepositFetcher:    depositCache,
			HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 0}},
		}

		ctx := context.Background()
		majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, beaconState)
		require.NoError(t, err)

		hash := majorityVoteEth1Data.BlockHash

		expectedHash := []byte("eth1data")
		assert.DeepEqual(t, expectedHash, hash)
	})
}

func TestProposer_FilterAttestation(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	genesis := util.NewBeaconBlock()

	numValidators := uint64(64)
	state, privKeys := util.DeterministicGenesisState(t, numValidators)
	require.NoError(t, state.SetGenesisValidatorRoot(params.BeaconConfig().ZeroHash[:]))
	assert.NoError(t, state.SetSlot(1))

	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		name         string
		wantedErr    string
		inputAtts    func() []*ethpb.Attestation
		expectedAtts func(inputAtts []*ethpb.Attestation) []*ethpb.Attestation
	}{
		{
			name: "nil attestations",
			inputAtts: func() []*ethpb.Attestation {
				return nil
			},
			expectedAtts: func(inputAtts []*ethpb.Attestation) []*ethpb.Attestation {
				return []*ethpb.Attestation{}
			},
		},
		{
			name: "invalid attestations",
			inputAtts: func() []*ethpb.Attestation {
				atts := make([]*ethpb.Attestation, 10)
				for i := 0; i < len(atts); i++ {
					atts[i] = util.HydrateAttestation(&ethpb.Attestation{
						Data: &ethpb.AttestationData{
							CommitteeIndex: types.CommitteeIndex(i),
						},
					})
				}
				return atts
			},
			expectedAtts: func(inputAtts []*ethpb.Attestation) []*ethpb.Attestation {
				return []*ethpb.Attestation{}
			},
		},
		{
			name: "filter aggregates ok",
			inputAtts: func() []*ethpb.Attestation {
				atts := make([]*ethpb.Attestation, 10)
				for i := 0; i < len(atts); i++ {
					atts[i] = util.HydrateAttestation(&ethpb.Attestation{
						Data: &ethpb.AttestationData{
							CommitteeIndex: types.CommitteeIndex(i),
							Source:         &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
						},
						AggregationBits: bitfield.Bitlist{0b00000110},
					})
					committee, err := helpers.BeaconCommitteeFromState(context.Background(), state, atts[i].Data.Slot, atts[i].Data.CommitteeIndex)
					assert.NoError(t, err)
					attestingIndices, err := attestation.AttestingIndices(atts[i].AggregationBits, committee)
					require.NoError(t, err)
					assert.NoError(t, err)
					domain, err := signing.Domain(state.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, params.BeaconConfig().ZeroHash[:])
					require.NoError(t, err)
					sigs := make([]bls.Signature, len(attestingIndices))
					zeroSig := [96]byte{}
					atts[i].Signature = zeroSig[:]

					for i, indice := range attestingIndices {
						hashTreeRoot, err := signing.ComputeSigningRoot(atts[i].Data, domain)
						require.NoError(t, err)
						sig := privKeys[indice].Sign(hashTreeRoot[:])
						sigs[i] = sig
					}
					atts[i].Signature = bls.AggregateSignatures(sigs).Marshal()
				}
				return atts
			},
			expectedAtts: func(inputAtts []*ethpb.Attestation) []*ethpb.Attestation {
				return []*ethpb.Attestation{inputAtts[0]}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposerServer := &Server{
				AttPool:     attestations.NewPool(),
				HeadFetcher: &mock.ChainService{State: state, Root: genesisRoot[:]},
			}
			atts := tt.inputAtts()
			received, err := proposerServer.validateAndDeleteAttsInPool(context.Background(), state, atts)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
				assert.Equal(t, nil, received)
			} else {
				assert.NoError(t, err)
				assert.DeepEqual(t, tt.expectedAtts(atts), received)
			}
		})
	}
}

func TestProposer_Deposits_ReturnsEmptyList_IfLatestEth1DataEqGenesisEth1Block(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
		GenesisEth1Block: height,
	}

	beaconState, err := v1.InitializeFromProto(&ethpb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:   bytesutil.PadTo([]byte("0x0"), 32),
			DepositRoot: make([]byte, 32),
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)
	blk := util.NewBeaconBlock()
	blk.Block.Slot = beaconState.Slot()
	blkRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*ethpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("a"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte("b"), 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*ethpb.DepositContainer
	for i := int64(2); i < 22; i++ {
		recentDeposits = append(recentDeposits, &ethpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             bytesutil.PadTo([]byte{byte(i)}, 48),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := dp.Deposit.Data.HashTreeRoot()
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		assert.NoError(t, depositTrie.Insert(depositHash[:], int(dp.Index)))
		assert.NoError(t, depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.HashTreeRoot()))
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.HashTreeRoot())
	}

	bs := &Server{
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err := bs.deposits(ctx, beaconState, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected number of pending deposits")
}

func TestProposer_DeleteAttsInPool_Aggregated(t *testing.T) {
	s := &Server{
		AttPool: attestations.NewPool(),
	}
	priv, err := bls.RandKey()
	require.NoError(t, err)
	sig := priv.Sign([]byte("foo")).Marshal()
	aggregatedAtts := []*ethpb.Attestation{
		util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b10101}, Signature: sig}),
		util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b11010}, Signature: sig})}
	unaggregatedAtts := []*ethpb.Attestation{
		util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b10010}, Signature: sig}),
		util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b10100}, Signature: sig})}

	require.NoError(t, s.AttPool.SaveAggregatedAttestations(aggregatedAtts))
	require.NoError(t, s.AttPool.SaveUnaggregatedAttestations(unaggregatedAtts))

	aa, err := attaggregation.Aggregate(aggregatedAtts)
	require.NoError(t, err)
	require.NoError(t, s.deleteAttsInPool(context.Background(), append(aa, unaggregatedAtts...)))
	assert.Equal(t, 0, len(s.AttPool.AggregatedAttestations()), "Did not delete aggregated attestation")
	atts, err := s.AttPool.UnaggregatedAttestations()
	require.NoError(t, err)
	assert.Equal(t, 0, len(atts), "Did not delete unaggregated attestation")
}

func TestProposer_GetBeaconBlock_PreForkEpoch(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := util.DeterministicGenesisState(t, 64)
	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	genBlk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       genesis.Block.Slot,
			ParentRoot: genesis.Block.ParentRoot,
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: genesis.Block.Body.RandaoReveal,
				Graffiti:     genesis.Block.Body.Graffiti,
				Eth1Data:     genesis.Block.Body.Eth1Data,
			},
		},
		Signature: genesis.Signature,
	}
	wsb := wrapper.WrappedPhase0SignedBeaconBlock(genBlk)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := &Server{
		HeadFetcher:       &mock.ChainService{State: beaconState, Root: parentRoot[:]},
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		BlockReceiver:     &mock.ChainService{},
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		MockEth1Votes:     true,
		AttPool:           attestations.NewPool(),
		SlashingsPool:     slashings.NewPool(),
		ExitPool:          voluntaryexits.NewPool(),
		StateGen:          stategen.New(db),
		SyncCommitteePool: synccommittee.NewStore(),
	}

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	req := &ethpb.BlockRequest{
		Slot:         1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}

	proposerSlashings := make([]*ethpb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := types.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
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
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			beaconState,
			privKeys[i+params.BeaconConfig().MaxProposerSlashings],
			types.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
		)
		require.NoError(t, err)
		attSlashings[i] = attesterSlashing
		err = proposerServer.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
		require.NoError(t, err)
	}
	block, err := proposerServer.GetBeaconBlock(ctx, req)
	require.NoError(t, err)
	phase0Blk, ok := block.GetBlock().(*ethpb.GenericBeaconBlock_Phase0)
	require.Equal(t, true, ok)

	assert.Equal(t, req.Slot, phase0Blk.Phase0.Slot)
	assert.DeepEqual(t, parentRoot[:], phase0Blk.Phase0.ParentRoot, "Expected block to have correct parent root")
	assert.DeepEqual(t, randaoReveal, phase0Blk.Phase0.Body.RandaoReveal, "Expected block to have correct randao reveal")
	assert.DeepEqual(t, req.Graffiti, phase0Blk.Phase0.Body.Graffiti, "Expected block to have correct Graffiti")
	assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(phase0Blk.Phase0.Body.ProposerSlashings)))
	assert.DeepEqual(t, proposerSlashings, phase0Blk.Phase0.Body.ProposerSlashings)
	assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(phase0Blk.Phase0.Body.AttesterSlashings)))
	assert.DeepEqual(t, attSlashings, phase0Blk.Phase0.Body.AttesterSlashings)
}

func TestProposer_GetBeaconBlock_PostForkEpoch(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	cfg := params.MainnetConfig().Copy()
	cfg.AltairForkEpoch = 1
	params.OverrideBeaconConfig(cfg)
	beaconState, privKeys := util.DeterministicGenesisState(t, 64)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	wsb := wrapper.WrappedPhase0SignedBeaconBlock(genesis)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, wsb), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	altairSlot, err := slots.EpochStart(params.BeaconConfig().AltairForkEpoch)
	require.NoError(t, err)

	genAltair := &ethpb.SignedBeaconBlockAltair{
		Block: &ethpb.BeaconBlockAltair{
			Slot:       altairSlot + 1,
			ParentRoot: parentRoot[:],
			StateRoot:  genesis.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyAltair{
				RandaoReveal:  genesis.Block.Body.RandaoReveal,
				Graffiti:      genesis.Block.Body.Graffiti,
				Eth1Data:      genesis.Block.Body.Eth1Data,
				SyncAggregate: &ethpb.SyncAggregate{SyncCommitteeBits: bitfield.NewBitvector512(), SyncCommitteeSignature: make([]byte, 96)},
			},
		},
		Signature: genesis.Signature,
	}

	blkRoot, err := genAltair.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, blkRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blkRoot), "Could not save genesis state")

	proposerServer := &Server{
		HeadFetcher:       &mock.ChainService{State: beaconState, Root: parentRoot[:]},
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		BlockReceiver:     &mock.ChainService{},
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		MockEth1Votes:     true,
		AttPool:           attestations.NewPool(),
		SlashingsPool:     slashings.NewPool(),
		ExitPool:          voluntaryexits.NewPool(),
		StateGen:          stategen.New(db),
		SyncCommitteePool: synccommittee.NewStore(),
	}

	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	require.NoError(t, err)
	req := &ethpb.BlockRequest{
		Slot:         altairSlot + 1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}

	proposerSlashings := make([]*ethpb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := types.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
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
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			beaconState,
			privKeys[i+params.BeaconConfig().MaxProposerSlashings],
			types.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
		)
		require.NoError(t, err)
		attSlashings[i] = attesterSlashing
		err = proposerServer.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
		require.NoError(t, err)
	}
	block, err := proposerServer.GetBeaconBlock(ctx, req)
	require.NoError(t, err)
	altairBlk, ok := block.GetBlock().(*ethpb.GenericBeaconBlock_Altair)
	require.Equal(t, true, ok)

	assert.Equal(t, req.Slot, altairBlk.Altair.Slot)
	assert.DeepEqual(t, parentRoot[:], altairBlk.Altair.ParentRoot, "Expected block to have correct parent root")
	assert.DeepEqual(t, randaoReveal, altairBlk.Altair.Body.RandaoReveal, "Expected block to have correct randao reveal")
	assert.DeepEqual(t, req.Graffiti, altairBlk.Altair.Body.Graffiti, "Expected block to have correct Graffiti")
	assert.Equal(t, params.BeaconConfig().MaxProposerSlashings, uint64(len(altairBlk.Altair.Body.ProposerSlashings)))
	assert.DeepEqual(t, proposerSlashings, altairBlk.Altair.Body.ProposerSlashings)
	assert.Equal(t, params.BeaconConfig().MaxAttesterSlashings, uint64(len(altairBlk.Altair.Body.AttesterSlashings)))
	assert.DeepEqual(t, attSlashings, altairBlk.Altair.Body.AttesterSlashings)
}

func TestProposer_GetSyncAggregate_OK(t *testing.T) {
	proposerServer := &Server{
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		SyncCommitteePool: synccommittee.NewStore(),
	}

	r := params.BeaconConfig().ZeroHash
	conts := []*ethpb.SyncCommitteeContribution{
		{Slot: 1, SubcommitteeIndex: 0, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b0001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 0, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 0, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1110}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 1, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b0001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 1, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 1, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1110}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 2, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b0001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 2, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 2, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1110}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 3, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b0001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 3, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1001}, BlockRoot: r[:]},
		{Slot: 1, SubcommitteeIndex: 3, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b1110}, BlockRoot: r[:]},
		{Slot: 2, SubcommitteeIndex: 0, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b10101010}, BlockRoot: r[:]},
		{Slot: 2, SubcommitteeIndex: 1, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b10101010}, BlockRoot: r[:]},
		{Slot: 2, SubcommitteeIndex: 2, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b10101010}, BlockRoot: r[:]},
		{Slot: 2, SubcommitteeIndex: 3, Signature: bls.NewAggregateSignature().Marshal(), AggregationBits: []byte{0b10101010}, BlockRoot: r[:]},
	}

	for _, cont := range conts {
		require.NoError(t, proposerServer.SyncCommitteePool.SaveSyncCommitteeContribution(cont))
	}

	aggregate, err := proposerServer.getSyncAggregate(context.Background(), 1, bytesutil.ToBytes32(conts[0].BlockRoot))
	require.NoError(t, err)
	require.DeepEqual(t, bitfield.Bitvector512{0xf, 0xf, 0xf, 0xf}, aggregate.SyncCommitteeBits)

	aggregate, err = proposerServer.getSyncAggregate(context.Background(), 2, bytesutil.ToBytes32(conts[0].BlockRoot))
	require.NoError(t, err)
	require.DeepEqual(t, bitfield.Bitvector512{0xaa, 0xaa, 0xaa, 0xaa}, aggregate.SyncCommitteeBits)

	aggregate, err = proposerServer.getSyncAggregate(context.Background(), 3, bytesutil.ToBytes32(conts[0].BlockRoot))
	require.NoError(t, err)
	require.DeepEqual(t, bitfield.NewBitvector512(), aggregate.SyncCommitteeBits)
}

func majorityVoteBoundaryTime(slot types.Slot) (uint64, uint64) {
	slots := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))
	slotStartTime := uint64(mockPOW.GenesisTime) + uint64((slot - (slot % (slots))).Mul(params.BeaconConfig().SecondsPerSlot))
	earliestValidTime := slotStartTime - 2*params.BeaconConfig().SecondsPerETH1Block*params.BeaconConfig().Eth1FollowDistance
	latestValidTime := slotStartTime - params.BeaconConfig().SecondsPerETH1Block*params.BeaconConfig().Eth1FollowDistance

	return earliestValidTime, latestValidTime
}

func BenchmarkGetBlock(b *testing.B) {
	proposerServer, beaconState, privKeys := setupGetBlock(b)
	ctx := context.Background()
	for n := 1; n < b.N; n++ {
		runGetBlock(ctx, b, proposerServer, beaconState, privKeys, n)
	}
}

func BenchmarkOptimisedGetBlock(b *testing.B) {
	// enable block optimisations flag
	resetCfg := features.InitWithReset(&features.Flags{
		EnableGetBlockOptimizations: true,
	})
	defer resetCfg()
	proposerServer, beaconState, privKeys := setupGetBlock(b)
	ctx := context.Background()
	for n := 1; n < b.N; n++ {
		runGetBlock(ctx, b, proposerServer, beaconState, privKeys, n)
	}
}

func runGetBlock(ctx context.Context, b *testing.B, proposerServer *Server, beaconState state.BeaconState, privKeys []bls.SecretKey, counter int) {
	randaoReveal, err := util.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(b, err)

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	req := &ethpb.BlockRequest{
		Slot:         types.Slot(counter),
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}

	_, err = proposerServer.GetBlock(ctx, req)
	require.NoError(b, err)
}

func setupGetBlock(bm *testing.B) (*Server, state.BeaconState, []bls.SecretKey) {
	db := dbutil.SetupDB(bm)
	ctx := context.Background()

	params.SetupTestConfigCleanup(bm)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := util.DeterministicGenesisState(bm, 64)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(bm, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	require.NoError(bm, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

	parentRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(bm, err, "Could not get signing root")
	require.NoError(bm, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(bm, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := &Server{
		HeadFetcher:       &mock.ChainService{State: beaconState, Root: parentRoot[:]},
		SyncChecker:       &mockSync.Sync{IsSyncing: false},
		BlockReceiver:     &mock.ChainService{},
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		MockEth1Votes:     true,
		AttPool:           attestations.NewPool(),
		SlashingsPool:     slashings.NewPool(),
		ExitPool:          voluntaryexits.NewPool(),
		StateGen:          stategen.New(db),
	}

	proposerSlashings := make([]*ethpb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings)
	for i := types.ValidatorIndex(0); uint64(i) < params.BeaconConfig().MaxProposerSlashings; i++ {
		proposerSlashing, err := util.GenerateProposerSlashingForValidator(
			beaconState,
			privKeys[i],
			i, /* validator index */
		)
		require.NoError(bm, err)
		proposerSlashings[i] = proposerSlashing
		err = proposerServer.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing)
		require.NoError(bm, err)
	}

	attSlashings := make([]*ethpb.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
		attesterSlashing, err := util.GenerateAttesterSlashingForValidator(
			beaconState,
			privKeys[i+params.BeaconConfig().MaxProposerSlashings],
			types.ValidatorIndex(i+params.BeaconConfig().MaxProposerSlashings), /* validator index */
		)
		require.NoError(bm, err)
		attSlashings[i] = attesterSlashing
		err = proposerServer.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing)
		require.NoError(bm, err)
	}
	return proposerServer, beaconState, privKeys
}
