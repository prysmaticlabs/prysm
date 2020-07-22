package validator

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	fastssz "github.com/ferranbt/fastssz"
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	attaggregation "github.com/prysmaticlabs/prysm/shared/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestGetBlock_OK(t *testing.T) {
	db, sc := dbutil.SetupDB(t)
	ctx := context.Background()

	testutil.ResetCache()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, genesis), "Could not save genesis block")

	parentRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := &Server{
		BeaconDB:          db,
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
		StateGen:          stategen.New(db, sc),
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
	for i := uint64(0); i < params.BeaconConfig().MaxProposerSlashings; i++ {
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
			i+params.BeaconConfig().MaxProposerSlashings, /* validator index */
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

func TestGetBlock_AddsUnaggregatedAtts(t *testing.T) {
	db, sc := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := testutil.DeterministicGenesisState(t, params.BeaconConfig().MinGenesisActiveValidatorCount)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, genesis), "Could not save genesis block")

	parentRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := &Server{
		BeaconDB:          db,
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
		StateGen:          stategen.New(db, sc),
	}

	// Generate a bunch of random attestations at slot. These would be considered double votes, but
	// we don't care for the purpose of this test.
	var atts []*ethpb.Attestation
	for i := uint64(0); len(atts) < int(params.BeaconConfig().MaxAttestations); i++ {
		a, err := testutil.GenerateAttestations(beaconState, privKeys, 2, 1, true)
		require.NoError(t, err)
		atts = append(atts, a...)
	}
	// Max attestations minus one so we can almost fill the block and then include 1 unaggregated
	// att to maximize inclusion.
	atts = atts[:params.BeaconConfig().MaxAttestations-1]
	require.NoError(t, proposerServer.AttPool.SaveAggregatedAttestations(atts))

	// Generate some more random attestations with a larger spread so that we can capture at least
	// one unaggregated attestation.
	if atts, err := testutil.GenerateAttestations(beaconState, privKeys, 300, 1, true); err != nil {
		t.Fatal(err)
	} else {
		found := false
		for _, a := range atts {
			if !helpers.IsAggregated(a) {
				found = true
				require.NoError(t, proposerServer.AttPool.SaveUnaggregatedAttestation(a))
			}
		}
		if !found {
			t.Fatal("No unaggregated attestations were generated")
		}
	}

	randaoReveal, err := testutil.RandaoReveal(beaconState, 0, privKeys)
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
	assert.Equal(t, params.BeaconConfig().MaxAttestations, uint64(len(block.Body.Attestations)), "Expected a full block of attestations")
	hasUnaggregatedAtt := false
	for _, a := range block.Body.Attestations {
		if !helpers.IsAggregated(a) {
			hasUnaggregatedAtt = true
			break
		}
	}
	assert.Equal(t, true, hasUnaggregatedAtt, "Expected block to contain at least one unaggregated attestation")
}

func TestProposeBlock_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	genesis := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(context.Background(), genesis), "Could not save genesis block")

	numDeposits := uint64(64)
	beaconState, _ := testutil.DeterministicGenesisState(t, numDeposits)
	bsRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

	c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
	proposerServer := &Server{
		BeaconDB:          db,
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		BlockReceiver:     c,
		HeadFetcher:       c,
		BlockNotifier:     c.BlockNotifier(),
		P2P:               mockp2p.NewTestP2P(t),
	}
	req := testutil.NewBeaconBlock()
	req.Block.Slot = 5
	req.Block.ParentRoot = bsRoot[:]
	require.NoError(t, db.SaveBlock(ctx, req))
	_, err = proposerServer.ProposeBlock(context.Background(), req)
	assert.NoError(t, err, "Could not propose block correctly")
}

func TestComputeStateRoot_OK(t *testing.T) {
	db, sc := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err, "Could not hash genesis state")

	genesis := b.NewGenesisBlock(stateRoot[:])
	require.NoError(t, db.SaveBlock(ctx, genesis), "Could not save genesis block")

	parentRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err, "Could not get signing root")
	require.NoError(t, db.SaveState(ctx, beaconState, parentRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, parentRoot), "Could not save genesis state")

	proposerServer := &Server{
		BeaconDB:          db,
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		StateGen:          stategen.New(db, sc),
	}
	req := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: 21,
			ParentRoot:    parentRoot[:],
			Slot:          1,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal:      nil,
				ProposerSlashings: nil,
				AttesterSlashings: nil,
				Eth1Data:          &ethpb.Eth1Data{},
			},
		},
	}
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+1))
	randaoReveal, err := testutil.RandaoReveal(beaconState, 0, privKeys)
	require.NoError(t, err)
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	require.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()-1))
	req.Block.Body.RandaoReveal = randaoReveal[:]
	currentEpoch := helpers.CurrentEpoch(beaconState)
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(req.Block, domain)
	require.NoError(t, err)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	req.Signature = blockSig[:]

	_, err = proposerServer.computeStateRoot(context.Background(), req)
	require.NoError(t, err)
}

func TestPendingDeposits_Eth1DataVoteOK(t *testing.T) {
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
	period := params.BeaconConfig().EpochsPerEth1VotingPeriod * params.BeaconConfig().SlotsPerEpoch
	for i := 0; i <= int(period/2); i++ {
		votes = append(votes, vote)
	}

	blockHash = make([]byte, 32)
	copy(blockHash, "0x0")
	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot:  make([]byte, 32),
			BlockHash:    blockHash,
			DepositCount: 2,
		},
		Eth1DepositIndex: 2,
		Eth1DataVotes:    votes,
	})
	require.NoError(t, err)

	blk := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{Eth1Data: &ethpb.Eth1Data{}},
	}

	blkRoot, err := ssz.HashTreeRoot(blk)
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

	newState, err := b.ProcessEth1DataInBlock(beaconState, blk)
	require.NoError(t, err)

	if proto.Equal(newState.Eth1Data(), vote) {
		t.Errorf("eth1data in the state equal to vote, when not expected to"+
			"have majority: Got %v", vote)
	}

	blk = &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{Eth1Data: vote},
	}

	_, eth1Height, err = bs.canonicalEth1Data(ctx, beaconState, vote)
	require.NoError(t, err)
	assert.Equal(t, 0, eth1Height.Cmp(newHeight))

	newState, err = b.ProcessEth1DataInBlock(beaconState, blk)
	require.NoError(t, err)

	if !proto.Equal(newState.Eth1Data(), vote) {
		t.Errorf("eth1data in the state not of the expected kind: Got %v but wanted %v", newState.Eth1Data(), vote)
	}
}

func TestPendingDeposits_OutsideEth1FollowWindow(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("0x0"),
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*dbpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 2,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 8,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*dbpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 400,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("c"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 600,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("d"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		depositTrie.Insert(depositHash[:], int(dp.Index))
		depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.Root())
	}

	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}

	blkRoot, err := ssz.HashTreeRoot(blk)
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

	deposits, err := bs.deposits(ctx, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected list of deposits")

	// It should not return the recent deposits after their follow window.
	// as latest block number makes no difference in retrieval of deposits
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err = bs.deposits(ctx, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected number of pending deposits")
}

func TestPendingDeposits_FollowsCorrectEth1Block(t *testing.T) {
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
		BlockHash:    []byte("0x1"),
		DepositCount: 7,
	}
	period := params.BeaconConfig().EpochsPerEth1VotingPeriod * params.BeaconConfig().SlotsPerEpoch
	for i := 0; i <= int(period/2); i++ {
		votes = append(votes, vote)
	}

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 5,
		},
		Eth1DepositIndex: 1,
		Eth1DataVotes:    votes,
	})
	require.NoError(t, err)
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}

	blkRoot, err := ssz.HashTreeRoot(blk)
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*dbpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 8,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 14,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*dbpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 5000,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("c"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 6000,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("d"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		depositTrie.Insert(depositHash[:], int(dp.Index))
		depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.Root())
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

	deposits, err := bs.deposits(ctx, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected list of deposits")

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	// we should get our pending deposits once this vote pushes the vote tally to include
	// the updated eth1 data.
	deposits, err = bs.deposits(ctx, vote)
	require.NoError(t, err)
	assert.Equal(t, len(recentDeposits), len(deposits), "Received unexpected number of pending deposits")
}

func TestPendingDeposits_CantReturnBelowStateEth1DepositIndex(t *testing.T) {
	ctx := context.Background()
	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 100,
		},
		Eth1DepositIndex: 10,
	})
	require.NoError(t, err)
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*dbpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*dbpb.DepositContainer
	for i := int64(2); i < 16; i++ {
		recentDeposits = append(recentDeposits, &dbpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte{byte(i)},
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		depositTrie.Insert(depositHash[:], int(dp.Index))
		depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.Root())
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
	deposits, err := bs.deposits(ctx, &ethpb.Eth1Data{})
	require.NoError(t, err)

	expectedDeposits := 6
	assert.Equal(t, expectedDeposits, len(deposits), "Received unexpected number of pending deposits")
}

func TestPendingDeposits_CantReturnMoreThanMax(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 100,
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	require.NoError(t, err)
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*dbpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*dbpb.DepositContainer
	for i := int64(2); i < 22; i++ {
		recentDeposits = append(recentDeposits, &dbpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte{byte(i)},
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		depositTrie.Insert(depositHash[:], int(dp.Index))
		depositCache.InsertDeposit(ctx, dp.Deposit, height.Uint64(), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, height.Uint64(), dp.Index, depositTrie.Root())
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
	deposits, err := bs.deposits(ctx, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().MaxDeposits, uint64(len(deposits)), "Received unexpected number of pending deposits")
}

func TestPendingDeposits_CantReturnMoreThanDepositCount(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 5,
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	require.NoError(t, err)
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*dbpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*dbpb.DepositContainer
	for i := int64(2); i < 22; i++ {
		recentDeposits = append(recentDeposits, &dbpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte{byte(i)},
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		depositTrie.Insert(depositHash[:], int(dp.Index))
		depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.Root())
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
	deposits, err := bs.deposits(ctx, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 3, len(deposits), "Received unexpected number of pending deposits")
}

func TestDepositTrie_UtilizesCachedFinalizedDeposits(t *testing.T) {
	ctx := context.Background()
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{EnableFinalizedDepositsCache: true})
	defer resetCfg()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 4,
		},
		Eth1DepositIndex: 1,
	})
	require.NoError(t, err)
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}

	blkRoot, err := ssz.HashTreeRoot(blk)
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	finalizedDeposits := []*dbpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 10,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*dbpb.DepositContainer{
		{
			Index:           2,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("c"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index:           3,
			Eth1BlockHeight: 11,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("d"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")
	for _, dp := range append(finalizedDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		depositTrie.Insert(depositHash[:], int(dp.Index))
		depositCache.InsertDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.Root())
	}
	depositCache.InsertFinalizedDeposits(ctx, 2)
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.Root())
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

	trie, err := bs.depositTrie(ctx, big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance)))
	require.NoError(t, err)

	actualRoot := trie.HashTreeRoot()
	expectedRoot := depositTrie.HashTreeRoot()
	assert.Equal(t, expectedRoot, actualRoot, "Incorrect deposit trie root")
}

func TestEth1Data_EmptyVotesFetchBlockHashFailure(t *testing.T) {
	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte{'a'},
		},
		Eth1DataVotes: []*ethpb.Eth1Data{},
	})
	require.NoError(t, err)
	p := &mockPOW.FaultyMockPOWChain{
		HashesByHeight: make(map[int][]byte),
	}
	proposerServer := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockReceiver:     &mock.ChainService{State: beaconState},
		HeadFetcher:       &mock.ChainService{State: beaconState},
	}
	_, err = proposerServer.eth1Data(context.Background(), beaconState.Slot()+1)
	assert.NoError(t, err, "A failed request should not have returned an error, got %v")
}

func TestDefaultEth1Data_NoBlockExists(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	deps := []*dbpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 8,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             make([]byte, 96),
					WithdrawalCredentials: make([]byte, 32),
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 14,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             make([]byte, 96),
					WithdrawalCredentials: make([]byte, 32),
				}},
		},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	for _, dp := range deps {
		depositCache.InsertDeposit(context.Background(), dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.Root())
	}

	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			0:  []byte("hash0"),
			12: []byte("hash12"),
		},
	}
	proposerServer := &Server{
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
	}

	defEth1Data := &ethpb.Eth1Data{
		DepositCount: 10,
		BlockHash:    []byte{'t', 'e', 's', 't'},
		DepositRoot:  []byte{'r', 'o', 'o', 't'},
	}

	p.Eth1Data = defEth1Data

	result, err := proposerServer.defaultEth1DataResponse(ctx, big.NewInt(16))
	require.NoError(t, err)

	if !proto.Equal(result, defEth1Data) {
		t.Errorf("Did not receive default eth1data. Wanted %v but Got %v", defEth1Data, result)
	}
}

func TestEth1Data(t *testing.T) {
	slot := uint64(20000)

	p := &mockPOW.POWChain{
		BlockNumberByTime: map[uint64]*big.Int{
			slot * params.BeaconConfig().SecondsPerSlot: big.NewInt(8196),
		},
		HashesByHeight: map[int][]byte{
			8180: []byte("8180"),
		},
		Eth1Data: &ethpb.Eth1Data{
			DepositCount: 55,
		},
	}

	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetEth1Data(&ethpb.Eth1Data{DepositCount: 55}))
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mock.ChainService{State: headState},
	}

	ctx := context.Background()
	eth1Data, err := ps.eth1Data(ctx, slot)
	require.NoError(t, err)
	assert.Equal(t, uint64(55), eth1Data.DepositCount)
}

func TestEth1Data_SmallerDepositCount(t *testing.T) {
	slot := uint64(20000)
	deps := []*dbpb.DepositContainer{
		{
			Index:           0,
			Eth1BlockHeight: 8,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             make([]byte, 96),
					WithdrawalCredentials: make([]byte, 32),
				}},
		},
		{
			Index:           1,
			Eth1BlockHeight: 14,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             make([]byte, 96),
					WithdrawalCredentials: make([]byte, 32),
				}},
		},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	for _, dp := range deps {
		depositCache.InsertDeposit(context.Background(), dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.Root())
	}

	p := &mockPOW.POWChain{
		BlockNumberByTime: map[uint64]*big.Int{
			slot * params.BeaconConfig().SecondsPerSlot: big.NewInt(4096),
		},
		HashesByHeight: map[int][]byte{
			4080: []byte("4080"),
		},
		Eth1Data: &ethpb.Eth1Data{
			DepositCount: 55,
		},
	}
	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		HeadFetcher:       &mock.ChainService{ETH1Data: &ethpb.Eth1Data{DepositCount: 10}},
		DepositFetcher:    depositCache,
	}

	ctx := context.Background()
	eth1Data, err := ps.eth1Data(ctx, slot)
	require.NoError(t, err)

	// Will default to 10 as the current deposit count in the
	// cache is only 2.
	assert.Equal(t, uint64(10), eth1Data.DepositCount)
}

func TestEth1Data_MockEnabled(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	// If a mock eth1 data votes is specified, we use the following for the
	// eth1data we provide to every proposer based on https://github.com/ethereum/eth2.0-pm/issues/62:
	//
	// slot_in_voting_period = current_slot % SLOTS_PER_ETH1_VOTING_PERIOD
	// Eth1Data(
	//   DepositRoot = hash(current_epoch + slot_in_voting_period),
	//   DepositCount = state.eth1_deposit_index,
	//   BlockHash = hash(hash(current_epoch + slot_in_voting_period)),
	// )
	ctx := context.Background()
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetEth1DepositIndex(64))
	ps := &Server{
		HeadFetcher:   &mock.ChainService{State: headState},
		BeaconDB:      db,
		MockEth1Votes: true,
	}
	headBlockRoot := [32]byte{1, 2, 3}
	require.NoError(t, db.SaveState(ctx, headState, headBlockRoot))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, headBlockRoot))

	eth1Data, err := ps.eth1Data(ctx, 100)
	require.NoError(t, err)
	period := params.BeaconConfig().EpochsPerEth1VotingPeriod * params.BeaconConfig().SlotsPerEpoch
	wantedSlot := 100 % period
	currentEpoch := helpers.SlotToEpoch(100)
	var enc []byte
	enc = fastssz.MarshalUint64(enc, currentEpoch+wantedSlot)
	depRoot := hashutil.Hash(enc)
	blockHash := hashutil.Hash(depRoot[:])
	want := &ethpb.Eth1Data{
		DepositRoot:  depRoot[:],
		BlockHash:    blockHash[:],
		DepositCount: 64,
	}
	if !proto.Equal(eth1Data, want) {
		t.Errorf("Wanted %v, received %v", want, eth1Data)
	}
}

func TestEth1DataMajorityVote_ChooseHighestCount(t *testing.T) {
	slot := uint64(64)

	p := &mockPOW.POWChain{
		BlockNumberByTime: map[uint64]*big.Int{
			32 * params.BeaconConfig().SecondsPerSlot: big.NewInt(50),
			64 * params.BeaconConfig().SecondsPerSlot: big.NewInt(100),
		},
		HashesByHeight: map[int][]byte{
			int(100 - params.BeaconConfig().Eth1FollowDistance - 1): []byte("first"),
			int(100 - params.BeaconConfig().Eth1FollowDistance):     []byte("second"),
		},
	}

	dc := dbpb.DepositContainer{
		Index:           0,
		Eth1BlockHeight: 0,
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             []byte("a"),
				Signature:             make([]byte, 96),
				WithdrawalCredentials: make([]byte, 32),
			}},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)
	depositCache.InsertDeposit(context.Background(), dc.Deposit, dc.Eth1BlockHeight, dc.Index, depositTrie.Root())

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{
			{BlockHash: []byte("first")},
			{BlockHash: []byte("first")},
			{BlockHash: []byte("second")},
		},
	})
	require.NoError(t, err)

	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockFetcher:      p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mock.ChainService{State: beaconState, ETH1Data: &ethpb.Eth1Data{}},
	}

	ctx := context.Background()
	majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, slot)
	require.NoError(t, err)

	hash := majorityVoteEth1Data.BlockHash

	expectedHash := []byte("first")
	if bytes.Compare(hash, expectedHash) != 0 {
		t.Errorf("Chosen eth1data for block hash %v vs expected %v", hash, expectedHash)
	}
}

func TestEth1DataMajorityVote_HighestCountBeforeRange_ChooseHighestCountWithinRange(t *testing.T) {
	slot := uint64(64)

	p := &mockPOW.POWChain{
		BlockNumberByTime: map[uint64]*big.Int{
			32 * params.BeaconConfig().SecondsPerSlot: big.NewInt(50),
			64 * params.BeaconConfig().SecondsPerSlot: big.NewInt(100),
		},
		HashesByHeight: map[int][]byte{
			int(50 - params.BeaconConfig().Eth1FollowDistance - 1):  []byte("before_range"),
			int(100 - params.BeaconConfig().Eth1FollowDistance - 1): []byte("first"),
			int(100 - params.BeaconConfig().Eth1FollowDistance):     []byte("second"),
		},
	}

	dc := dbpb.DepositContainer{
		Index:           0,
		Eth1BlockHeight: 0,
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             []byte("a"),
				Signature:             make([]byte, 96),
				WithdrawalCredentials: make([]byte, 32),
			}},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)
	depositCache.InsertDeposit(context.Background(), dc.Deposit, dc.Eth1BlockHeight, dc.Index, depositTrie.Root())

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{
			{BlockHash: []byte("before_range")},
			{BlockHash: []byte("before_range")},
			{BlockHash: []byte("first")},
		},
	})
	require.NoError(t, err)

	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockFetcher:      p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mock.ChainService{State: beaconState, ETH1Data: &ethpb.Eth1Data{}},
	}

	ctx := context.Background()
	majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, slot)
	require.NoError(t, err)

	hash := majorityVoteEth1Data.BlockHash

	expectedHash := []byte("first")
	if bytes.Compare(hash, expectedHash) != 0 {
		t.Errorf("Chosen eth1data for block hash %v vs expected %v", hash, expectedHash)
	}
}

func TestEth1DataMajorityVote_HighestCountAfterRange_ChooseHighestCountWithinRange(t *testing.T) {
	slot := uint64(64)

	p := &mockPOW.POWChain{
		BlockNumberByTime: map[uint64]*big.Int{
			32 * params.BeaconConfig().SecondsPerSlot: big.NewInt(50),
			64 * params.BeaconConfig().SecondsPerSlot: big.NewInt(100),
		},
		HashesByHeight: map[int][]byte{
			int(100 - params.BeaconConfig().Eth1FollowDistance):     []byte("first"),
			int(100 - params.BeaconConfig().Eth1FollowDistance + 1): []byte("after_range"),
		},
	}

	dc := dbpb.DepositContainer{
		Index:           0,
		Eth1BlockHeight: 0,
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             []byte("a"),
				Signature:             make([]byte, 96),
				WithdrawalCredentials: make([]byte, 32),
			}},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)
	depositCache.InsertDeposit(context.Background(), dc.Deposit, dc.Eth1BlockHeight, dc.Index, depositTrie.Root())

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{
			{BlockHash: []byte("first")},
			{BlockHash: []byte("after_range")},
			{BlockHash: []byte("after_range")},
		},
	})
	require.NoError(t, err)

	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockFetcher:      p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mock.ChainService{State: beaconState, ETH1Data: &ethpb.Eth1Data{}},
	}

	ctx := context.Background()
	majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, slot)
	require.NoError(t, err)

	hash := majorityVoteEth1Data.BlockHash

	expectedHash := []byte("first")
	if bytes.Compare(hash, expectedHash) != 0 {
		t.Errorf("Chosen eth1data for block hash %v vs expected %v", hash, expectedHash)
	}
}

func TestEth1DataMajorityVote_HighestCountOnUnknownBlock_ChooseKnownBlockWithHighestCount(t *testing.T) {
	slot := uint64(64)

	p := &mockPOW.POWChain{
		BlockNumberByTime: map[uint64]*big.Int{
			32 * params.BeaconConfig().SecondsPerSlot: big.NewInt(50),
			64 * params.BeaconConfig().SecondsPerSlot: big.NewInt(100),
		},
		HashesByHeight: map[int][]byte{
			int(100 - params.BeaconConfig().Eth1FollowDistance - 1): []byte("first"),
			int(100 - params.BeaconConfig().Eth1FollowDistance):     []byte("second"),
		},
	}

	dc := dbpb.DepositContainer{
		Index:           0,
		Eth1BlockHeight: 0,
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             []byte("a"),
				Signature:             make([]byte, 96),
				WithdrawalCredentials: make([]byte, 32),
			}},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)
	depositCache.InsertDeposit(context.Background(), dc.Deposit, dc.Eth1BlockHeight, dc.Index, depositTrie.Root())

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{
			{BlockHash: []byte("unknown")},
			{BlockHash: []byte("unknown")},
			{BlockHash: []byte("first")},
		},
	})
	require.NoError(t, err)

	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockFetcher:      p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mock.ChainService{State: beaconState, ETH1Data: &ethpb.Eth1Data{}},
	}

	ctx := context.Background()
	majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, slot)
	require.NoError(t, err)

	hash := majorityVoteEth1Data.BlockHash

	expectedHash := []byte("first")
	if bytes.Compare(hash, expectedHash) != 0 {
		t.Errorf("Chosen eth1data for block hash %v vs expected %v", hash, expectedHash)
	}
}

func TestEth1DataMajorityVote_NoVotesInRange_ChooseDefault(t *testing.T) {
	slot := uint64(64)

	p := &mockPOW.POWChain{
		BlockNumberByTime: map[uint64]*big.Int{
			32 * params.BeaconConfig().SecondsPerSlot: big.NewInt(50),
			64 * params.BeaconConfig().SecondsPerSlot: big.NewInt(100),
		},
		HashesByHeight: map[int][]byte{
			int(50 - params.BeaconConfig().Eth1FollowDistance - 1):  []byte("before_range"),
			int(100 - params.BeaconConfig().Eth1FollowDistance - 1): []byte("first"),
			int(100 - params.BeaconConfig().Eth1FollowDistance):     []byte("second"),
			int(100 - params.BeaconConfig().Eth1FollowDistance + 1): []byte("after_range"),
		},
	}

	dc := dbpb.DepositContainer{
		Index:           0,
		Eth1BlockHeight: 0,
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             []byte("a"),
				Signature:             make([]byte, 96),
				WithdrawalCredentials: make([]byte, 32),
			}},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)
	depositCache.InsertDeposit(context.Background(), dc.Deposit, dc.Eth1BlockHeight, dc.Index, depositTrie.Root())

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{
			{BlockHash: []byte("before_range")},
			{BlockHash: []byte("after_range")},
		},
	})
	require.NoError(t, err)

	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockFetcher:      p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mock.ChainService{State: beaconState, ETH1Data: &ethpb.Eth1Data{}},
	}

	ctx := context.Background()
	majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, slot)
	require.NoError(t, err)

	hash := majorityVoteEth1Data.BlockHash

	expectedHash := make([]byte, 32)
	copy(expectedHash, "second")
	if bytes.Compare(hash, expectedHash) != 0 {
		t.Errorf("Chosen eth1data for block hash %v vs expected %v", hash, expectedHash)
	}
}

func TestEth1DataMajorityVote_NoVotes_ChooseDefault(t *testing.T) {
	slot := uint64(64)

	p := &mockPOW.POWChain{
		BlockNumberByTime: map[uint64]*big.Int{
			32 * params.BeaconConfig().SecondsPerSlot: big.NewInt(50),
			64 * params.BeaconConfig().SecondsPerSlot: big.NewInt(100),
		},
		HashesByHeight: map[int][]byte{
			int(100 - params.BeaconConfig().Eth1FollowDistance - 1): []byte("first"),
			int(100 - params.BeaconConfig().Eth1FollowDistance):     []byte("second"),
		},
	}

	dc := dbpb.DepositContainer{
		Index:           0,
		Eth1BlockHeight: 0,
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             []byte("a"),
				Signature:             make([]byte, 96),
				WithdrawalCredentials: make([]byte, 32),
			}},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)
	depositCache.InsertDeposit(context.Background(), dc.Deposit, dc.Eth1BlockHeight, dc.Index, depositTrie.Root())

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{Eth1DataVotes: []*ethpb.Eth1Data{}})
	require.NoError(t, err)

	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockFetcher:      p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mock.ChainService{State: beaconState, ETH1Data: &ethpb.Eth1Data{}},
	}

	ctx := context.Background()
	majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, slot)
	require.NoError(t, err)

	hash := majorityVoteEth1Data.BlockHash

	expectedHash := make([]byte, 32)
	copy(expectedHash, "second")
	if bytes.Compare(hash, expectedHash) != 0 {
		t.Errorf("Chosen eth1data for block hash %v vs expected %v", hash, expectedHash)
	}
}

func TestEth1DataMajorityVote_SameCount_ChooseMoreRecentBlock(t *testing.T) {
	slot := uint64(64)

	p := &mockPOW.POWChain{
		BlockNumberByTime: map[uint64]*big.Int{
			32 * params.BeaconConfig().SecondsPerSlot: big.NewInt(50),
			64 * params.BeaconConfig().SecondsPerSlot: big.NewInt(100),
		},
		HashesByHeight: map[int][]byte{
			int(100 - params.BeaconConfig().Eth1FollowDistance - 1): []byte("first"),
			int(100 - params.BeaconConfig().Eth1FollowDistance):     []byte("second"),
		},
	}

	dc := dbpb.DepositContainer{
		Index:           0,
		Eth1BlockHeight: 0,
		Deposit: &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey:             []byte("a"),
				Signature:             make([]byte, 96),
				WithdrawalCredentials: make([]byte, 32),
			}},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err)
	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)
	depositCache.InsertDeposit(context.Background(), dc.Deposit, dc.Eth1BlockHeight, dc.Index, depositTrie.Root())

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{
			{BlockHash: []byte("first")},
			{BlockHash: []byte("second")},
		},
	})
	require.NoError(t, err)

	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockFetcher:      p,
		DepositFetcher:    depositCache,
		HeadFetcher:       &mock.ChainService{State: beaconState, ETH1Data: &ethpb.Eth1Data{}},
	}

	ctx := context.Background()
	majorityVoteEth1Data, err := ps.eth1DataMajorityVote(ctx, slot)
	require.NoError(t, err)

	hash := majorityVoteEth1Data.BlockHash

	expectedHash := []byte("second")
	if bytes.Compare(hash, expectedHash) != 0 {
		t.Errorf("Chosen eth1data for block hash %v vs expected %v", hash, expectedHash)
	}
}

func TestFilterAttestation_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	genesis := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(context.Background(), genesis), "Could not save genesis block")

	numDeposits := uint64(64)
	state, privKeys := testutil.DeterministicGenesisState(t, numDeposits)
	require.NoError(t, state.SetGenesisValidatorRoot(params.BeaconConfig().ZeroHash[:]))
	assert.NoError(t, state.SetSlot(1))

	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, state, genesisRoot), "Could not save genesis state")
	require.NoError(t, db.SaveHeadBlockRoot(ctx, genesisRoot), "Could not save genesis state")

	proposerServer := &Server{
		BeaconDB:    db,
		AttPool:     attestations.NewPool(),
		HeadFetcher: &mock.ChainService{State: state, Root: genesisRoot[:]},
	}

	atts := make([]*ethpb.Attestation, 10)
	for i := 0; i < len(atts); i++ {
		atts[i] = &ethpb.Attestation{Data: &ethpb.AttestationData{
			CommitteeIndex: uint64(i),
			Target:         &ethpb.Checkpoint{}},
		}
	}
	received, err := proposerServer.filterAttestationsForBlockInclusion(context.Background(), state, atts)
	require.NoError(t, err)
	if len(received) > 0 {
		t.Error("Invalid attestations were filtered")
	}

	for i := 0; i < len(atts); i++ {
		aggBits := bitfield.NewBitlist(2)
		aggBits.SetBitAt(0, true)
		atts[i] = &ethpb.Attestation{Data: &ethpb.AttestationData{
			CommitteeIndex: uint64(i),
			Target:         &ethpb.Checkpoint{},
			Source:         &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}},
			AggregationBits: aggBits,
		}
		committee, err := helpers.BeaconCommitteeFromState(state, atts[i].Data.Slot, atts[i].Data.CommitteeIndex)
		assert.NoError(t, err)
		attestingIndices := attestationutil.AttestingIndices(atts[i].AggregationBits, committee)
		assert.NoError(t, err)
		domain, err := helpers.Domain(state.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, params.BeaconConfig().ZeroHash[:])
		require.NoError(t, err)
		sigs := make([]bls.Signature, len(attestingIndices))
		zeroSig := [96]byte{}
		atts[i].Signature = zeroSig[:]

		for i, indice := range attestingIndices {
			hashTreeRoot, err := helpers.ComputeSigningRoot(atts[i].Data, domain)
			require.NoError(t, err)
			sig := privKeys[indice].Sign(hashTreeRoot[:])
			sigs[i] = sig
		}
		atts[i].Signature = bls.AggregateSignatures(sigs).Marshal()[:]
	}

	received, err = proposerServer.filterAttestationsForBlockInclusion(context.Background(), state, atts)
	require.NoError(t, err)
	assert.Equal(t, 1, len(received), "Did not filter correctly")
}

func Benchmark_Eth1Data(b *testing.B) {
	ctx := context.Background()

	hashesByHeight := make(map[int][]byte)

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{},
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("stub"),
		},
	})
	require.NoError(b, err)
	var mockSig [96]byte
	var mockCreds [32]byte
	deposits := []*dbpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(b, err)

	for i, dp := range deposits {
		var root [32]byte
		copy(root[:], []byte{'d', 'e', 'p', 'o', 's', 'i', 't', byte(i)})
		depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, root)
	}
	numOfVotes := 1000
	for i := 0; i < numOfVotes; i++ {
		blockhash := []byte{'b', 'l', 'o', 'c', 'k', byte(i)}
		deposit := []byte{'d', 'e', 'p', 'o', 's', 'i', 't', byte(i)}
		err := beaconState.SetEth1DataVotes(append(beaconState.Eth1DataVotes(), &ethpb.Eth1Data{
			BlockHash:   blockhash,
			DepositRoot: deposit,
		}))
		require.NoError(b, err)
		hashesByHeight[i] = blockhash
	}
	hashesByHeight[numOfVotes+1] = []byte("stub")

	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	require.NoError(b, err)

	currentHeight := params.BeaconConfig().Eth1FollowDistance + 5
	p := &mockPOW.POWChain{
		LatestBlockNumber: big.NewInt(int64(currentHeight)),
		HashesByHeight:    hashesByHeight,
	}
	proposerServer := &Server{
		BlockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		HeadFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		ChainStartFetcher:      p,
		Eth1InfoFetcher:        p,
		Eth1BlockFetcher:       p,
		DepositFetcher:         depositCache,
		PendingDepositsFetcher: depositCache,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proposerServer.eth1Data(context.Background(), beaconState.Slot()+1)
		require.NoError(b, err)
	}
}

func TestDeposits_ReturnsEmptyList_IfLatestEth1DataEqGenesisEth1Block(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
		GenesisEth1Block: height,
	}

	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("0x0"),
		},
		Eth1DepositIndex: 2,
	})
	require.NoError(t, err)
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	require.NoError(t, err)

	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*dbpb.DepositContainer{
		{
			Index: 0,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*dbpb.DepositContainer
	for i := int64(2); i < 22; i++ {
		recentDeposits = append(recentDeposits, &dbpb.DepositContainer{
			Index: i,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte{byte(i)},
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		})
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	require.NoError(t, err, "Could not setup deposit trie")

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		require.NoError(t, err, "Unable to determine hashed value of deposit")

		depositTrie.Insert(depositHash[:], int(dp.Index))
		depositCache.InsertDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, uint64(dp.Index), dp.Index, depositTrie.Root())
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
	deposits, err := bs.deposits(ctx, &ethpb.Eth1Data{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(deposits), "Received unexpected number of pending deposits")
}

func TestDeleteAttsInPool_Aggregated(t *testing.T) {
	s := &Server{
		AttPool: attestations.NewPool(),
	}

	sig := bls.RandKey().Sign([]byte("foo")).Marshal()
	aggregatedAtts := []*ethpb.Attestation{{AggregationBits: bitfield.Bitlist{0b10101}, Signature: sig}, {AggregationBits: bitfield.Bitlist{0b11010}, Signature: sig}}
	unaggregatedAtts := []*ethpb.Attestation{{AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig}, {AggregationBits: bitfield.Bitlist{0b0001}, Signature: sig}}

	require.NoError(t, s.AttPool.SaveAggregatedAttestations(aggregatedAtts))
	require.NoError(t, s.AttPool.SaveUnaggregatedAttestations(unaggregatedAtts))

	aa, err := attaggregation.Aggregate(aggregatedAtts)
	require.NoError(t, err)
	require.NoError(t, s.deleteAttsInPool(context.Background(), append(aa, unaggregatedAtts...)))
	assert.Equal(t, 0, len(s.AttPool.AggregatedAttestations()), "Did not delete aggregated attestation")
	assert.Equal(t, 0, len(s.AttPool.UnaggregatedAttestations()), "Did not delete unaggregated attestation")
}
