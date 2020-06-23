package validator

import (
	"bytes"
	"context"
	"math/big"
	"reflect"
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
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	if err != nil {
		t.Fatalf("Could not hash genesis state: %v", err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	if err := db.SaveBlock(ctx, genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	parentRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	if err := db.SaveState(ctx, beaconState, parentRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, parentRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

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
	if err != nil {
		t.Error(err)
	}

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
		if err != nil {
			t.Fatal(err)
		}
		proposerSlashings[i] = proposerSlashing
		if err := proposerServer.SlashingsPool.InsertProposerSlashing(context.Background(), beaconState, proposerSlashing); err != nil {
			t.Fatal(err)
		}
	}

	attSlashings := make([]*ethpb.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings)
	for i := uint64(0); i < params.BeaconConfig().MaxAttesterSlashings; i++ {
		attesterSlashing, err := testutil.GenerateAttesterSlashingForValidator(
			beaconState,
			privKeys[i+params.BeaconConfig().MaxProposerSlashings],
			i+params.BeaconConfig().MaxProposerSlashings, /* validator index */
		)
		if err != nil {
			t.Fatal(err)
		}
		attSlashings[i] = attesterSlashing
		if err := proposerServer.SlashingsPool.InsertAttesterSlashing(context.Background(), beaconState, attesterSlashing); err != nil {
			t.Fatal(err)
		}
	}

	block, err := proposerServer.GetBlock(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	if block.Slot != req.Slot {
		t.Fatal("Expected block to have slot of 1")
	}
	if !bytes.Equal(block.ParentRoot, parentRoot[:]) {
		t.Fatal("Expected block to have correct parent root")
	}
	if !bytes.Equal(block.Body.RandaoReveal, randaoReveal) {
		t.Fatal("Expected block to have correct randao reveal")
	}
	if !bytes.Equal(block.Body.Graffiti, req.Graffiti) {
		t.Fatal("Expected block to have correct graffiti")
	}
	if uint64(len(block.Body.ProposerSlashings)) != params.BeaconConfig().MaxProposerSlashings {
		t.Fatalf("Wanted %d proposer slashings, got %d", params.BeaconConfig().MaxProposerSlashings, len(block.Body.ProposerSlashings))
	}
	if !reflect.DeepEqual(block.Body.ProposerSlashings, proposerSlashings) {
		t.Errorf("Wanted proposer slashing %v, got %v", proposerSlashings, block.Body.ProposerSlashings)
	}
	if uint64(len(block.Body.AttesterSlashings)) != params.BeaconConfig().MaxAttesterSlashings {
		t.Fatalf("Wanted %d attester slashings, got %d", params.BeaconConfig().MaxAttesterSlashings, len(block.Body.AttesterSlashings))
	}
	if !reflect.DeepEqual(block.Body.AttesterSlashings, attSlashings) {
		t.Errorf("Wanted attester slashing %v, got %v", attSlashings, block.Body.AttesterSlashings)
	}
}

func TestGetBlock_AddsUnaggregatedAtts(t *testing.T) {
	db, sc := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := testutil.DeterministicGenesisState(t, params.BeaconConfig().MinGenesisActiveValidatorCount)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	if err != nil {
		t.Fatalf("Could not hash genesis state: %v", err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	if err := db.SaveBlock(ctx, genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	parentRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	if err := db.SaveState(ctx, beaconState, parentRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, parentRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

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
		if err != nil {
			t.Fatal(err)
		}
		atts = append(atts, a...)
	}
	// Max attestations minus one so we can almost fill the block and then include 1 unaggregated
	// att to maximize inclusion.
	atts = atts[:params.BeaconConfig().MaxAttestations-1]
	if err := proposerServer.AttPool.SaveAggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}

	// Generate some more random attestations with a larger spread so that we can capture at least
	// one unaggregated attestation.
	if atts, err := testutil.GenerateAttestations(beaconState, privKeys, 300, 1, true); err != nil {
		t.Fatal(err)
	} else {
		found := false
		for _, a := range atts {
			if !helpers.IsAggregated(a) {
				found = true
				if err := proposerServer.AttPool.SaveUnaggregatedAttestation(a); err != nil {
					t.Fatal(err)
				}
			}
		}
		if !found {
			t.Fatal("No unaggregated attestations were generated")
		}
	}

	randaoReveal, err := testutil.RandaoReveal(beaconState, 0, privKeys)
	if err != nil {
		t.Error(err)
	}

	graffiti := bytesutil.ToBytes32([]byte("eth2"))
	req := &ethpb.BlockRequest{
		Slot:         1,
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti[:],
	}

	block, err := proposerServer.GetBlock(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	if block.Slot != req.Slot {
		t.Fatal("Expected block to have slot of 1")
	}
	if !bytes.Equal(block.ParentRoot, parentRoot[:]) {
		t.Fatal("Expected block to have correct parent root")
	}
	if !bytes.Equal(block.Body.RandaoReveal, randaoReveal) {
		t.Fatal("Expected block to have correct randao reveal")
	}
	if !bytes.Equal(block.Body.Graffiti, req.Graffiti) {
		t.Fatal("Expected block to have correct graffiti")
	}
	if uint64(len(block.Body.Attestations)) != params.BeaconConfig().MaxAttestations {
		t.Fatalf("Expected a full block of attestations, only received %d", len(block.Body.Attestations))
	}
	hasUnaggregatedAtt := false
	for _, a := range block.Body.Attestations {
		if !helpers.IsAggregated(a) {
			hasUnaggregatedAtt = true
			break
		}
	}
	if !hasUnaggregatedAtt {
		t.Fatal("Expected block to contain at least one unaggregated attestation")
	}
}

func TestProposeBlock_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	genesis := testutil.NewBeaconBlock()
	if err := db.SaveBlock(context.Background(), genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	numDeposits := uint64(64)
	beaconState, _ := testutil.DeterministicGenesisState(t, numDeposits)

	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, genesisRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	c := &mock.ChainService{}
	proposerServer := &Server{
		BeaconDB:          db,
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		BlockReceiver:     c,
		HeadFetcher:       c,
		BlockNotifier:     c.BlockNotifier(),
	}
	req := testutil.NewBeaconBlock()
	req.Block.Slot = 5
	req.Block.ParentRoot = bytesutil.PadTo([]byte("parent-hash"), 32)
	if err := db.SaveBlock(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := proposerServer.ProposeBlock(context.Background(), req); err != nil {
		t.Errorf("Could not propose block correctly: %v", err)
	}
}

func TestComputeStateRoot_OK(t *testing.T) {
	db, sc := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	stateRoot, err := beaconState.HashTreeRoot(ctx)
	if err != nil {
		t.Fatalf("Could not hash genesis state: %v", err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	if err := db.SaveBlock(ctx, genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	parentRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	if err := db.SaveState(ctx, beaconState, parentRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, parentRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

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
	if err := beaconState.SetSlot(beaconState.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	randaoReveal, err := testutil.RandaoReveal(beaconState, 0, privKeys)
	if err != nil {
		t.Error(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}
	if err := beaconState.SetSlot(beaconState.Slot() - 1); err != nil {
		t.Fatal(err)
	}
	req.Block.Body.RandaoReveal = randaoReveal[:]
	currentEpoch := helpers.CurrentEpoch(beaconState)
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(req.Block, domain)
	if err != nil {
		t.Error(err)
	}
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	req.Signature = blockSig[:]

	_, err = proposerServer.computeStateRoot(context.Background(), req)
	if err != nil {
		t.Error(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	blk := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{Eth1Data: &ethpb.Eth1Data{}},
	}

	blkRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	if eth1Height.Cmp(height) != 0 {
		t.Errorf("Wanted Eth1 height of %d but got %d", height.Uint64(), eth1Height.Uint64())
	}

	newState, err := b.ProcessEth1DataInBlock(beaconState, blk)
	if err != nil {
		t.Fatal(err)
	}

	if proto.Equal(newState.Eth1Data(), vote) {
		t.Errorf("eth1data in the state equal to vote, when not expected to"+
			"have majority: Got %v", vote)
	}

	blk = &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{Eth1Data: vote},
	}

	_, eth1Height, err = bs.canonicalEth1Data(ctx, beaconState, vote)
	if err != nil {
		t.Fatal(err)
	}

	if eth1Height.Cmp(newHeight) != 0 {
		t.Errorf("Wanted Eth1 height of %d but got %d", newHeight.Uint64(), eth1Height.Uint64())
	}

	newState, err = b.ProcessEth1DataInBlock(beaconState, blk)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

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
	depositCache := depositcache.NewDepositCache()
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		if err != nil {
			t.Fatalf("Unable to determine hashed value of deposit %v", err)
		}

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
	if err != nil {
		t.Fatal(err)
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
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != 0 {
		t.Errorf("Received unexpected list of deposits: %+v, wanted: 0", len(deposits))
	}

	// It should not return the recent deposits after their follow window.
	// as latest block number makes no difference in retrieval of deposits
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	deposits, err = bs.deposits(ctx, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != 0 {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: %d",
			len(deposits),
			len(recentDeposits),
		)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}

	blkRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

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
	depositCache := depositcache.NewDepositCache()
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		if err != nil {
			t.Fatalf("Unable to determine hashed value of deposit %v", err)
		}

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
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != 0 {
		t.Errorf("Received unexpected list of deposits: %+v, wanted: 0", len(deposits))
	}

	// It should also return the recent deposits after their follow window.
	p.LatestBlockNumber = big.NewInt(0).Add(p.LatestBlockNumber, big.NewInt(10000))
	// we should get our pending deposits once this vote pushes the vote tally to include
	// the updated eth1 data.
	deposits, err = bs.deposits(ctx, vote)
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != len(recentDeposits) {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: %d",
			len(deposits),
			len(recentDeposits),
		)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	depositCache := depositcache.NewDepositCache()
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		if err != nil {
			t.Fatalf("Unable to determine hashed value of deposit %v", err)
		}

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
	if err != nil {
		t.Fatal(err)
	}

	expectedDeposits := 6
	if len(deposits) != expectedDeposits {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: %d",
			len(deposits),
			expectedDeposits,
		)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	depositCache := depositcache.NewDepositCache()
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		if err != nil {
			t.Fatalf("Unable to determine hashed value of deposit %v", err)
		}

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
	if err != nil {
		t.Fatal(err)
	}
	if uint64(len(deposits)) != params.BeaconConfig().MaxDeposits {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: %d",
			len(deposits),
			int(params.BeaconConfig().MaxDeposits),
		)
	}
}

func TestPendingDeposits_CantReturnMoreDepositCount(t *testing.T) {
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
	if err != nil {
		t.Fatal(err)
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	depositCache := depositcache.NewDepositCache()
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		if err != nil {
			t.Fatalf("Unable to determine hashed value of deposit %v", err)
		}

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
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != 3 {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: %d",
			len(deposits),
			3,
		)
	}
}

func TestEth1Data_EmptyVotesFetchBlockHashFailure(t *testing.T) {
	beaconState, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte{'a'},
		},
		Eth1DataVotes: []*ethpb.Eth1Data{},
	})
	if err != nil {
		t.Fatal(err)
	}
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
	if _, err := proposerServer.eth1Data(context.Background(), beaconState.Slot()+1); err != nil {
		t.Errorf("A failed request should not have returned an error, got %v", err)
	}
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
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	depositCache := depositcache.NewDepositCache()
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
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(result, defEth1Data) {
		t.Errorf("Did not receive default eth1data. Wanted %v but Got %v", defEth1Data, result)
	}
}

// TODO(2312): Add more tests for edge cases and better coverage.
func TestEth1Data(t *testing.T) {
	slot := uint64(20000)

	p := &mockPOW.POWChain{
		BlockNumberByHeight: map[uint64]*big.Int{
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
	if err := headState.SetEth1Data(&ethpb.Eth1Data{DepositCount: 55}); err != nil {
		t.Fatal(err)
	}
	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		DepositFetcher:    depositcache.NewDepositCache(),
		HeadFetcher:       &mock.ChainService{State: headState},
	}

	ctx := context.Background()
	eth1Data, err := ps.eth1Data(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}

	if eth1Data.DepositCount != 55 {
		t.Error("Expected deposit count to be 55")
	}
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
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	depositCache := depositcache.NewDepositCache()
	for _, dp := range deps {
		depositCache.InsertDeposit(context.Background(), dp.Deposit, dp.Eth1BlockHeight, dp.Index, depositTrie.Root())
	}

	p := &mockPOW.POWChain{
		BlockNumberByHeight: map[uint64]*big.Int{
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
	if err != nil {
		t.Fatal(err)
	}

	// Will default to 10 as the current deposit count in the
	// cache is only 2.
	if eth1Data.DepositCount != 10 {
		t.Errorf("Expected deposit count to be 10 but got %d", eth1Data.DepositCount)
	}
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
	if err := headState.SetEth1DepositIndex(64); err != nil {
		t.Fatal(err)
	}
	ps := &Server{
		HeadFetcher:   &mock.ChainService{State: headState},
		BeaconDB:      db,
		MockEth1Votes: true,
	}
	headBlockRoot := [32]byte{1, 2, 3}
	if err := db.SaveState(ctx, headState, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, headBlockRoot); err != nil {
		t.Fatal(err)
	}

	eth1Data, err := ps.eth1Data(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
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

func TestFilterAttestation_OK(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	genesis := testutil.NewBeaconBlock()
	if err := db.SaveBlock(context.Background(), genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	numDeposits := uint64(64)
	state, privKeys := testutil.DeterministicGenesisState(t, numDeposits)
	if err := state.SetGenesisValidatorRoot(params.BeaconConfig().ZeroHash[:]); err != nil {
		t.Fatal(err)
	}
	if err := state.SetSlot(1); err != nil {
		t.Error(err)
	}

	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, state, genesisRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, genesisRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

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
	if err != nil {
		t.Fatal(err)
	}
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
		if err != nil {
			t.Error(err)
		}
		attestingIndices := attestationutil.AttestingIndices(atts[i].AggregationBits, committee)
		if err != nil {
			t.Error(err)
		}
		domain, err := helpers.Domain(state.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, params.BeaconConfig().ZeroHash[:])
		if err != nil {
			t.Fatal(err)
		}
		sigs := make([]*bls.Signature, len(attestingIndices))
		zeroSig := [96]byte{}
		atts[i].Signature = zeroSig[:]

		for i, indice := range attestingIndices {
			hashTreeRoot, err := helpers.ComputeSigningRoot(atts[i].Data, domain)
			if err != nil {
				t.Fatal(err)
			}
			sig := privKeys[indice].Sign(hashTreeRoot[:])
			sigs[i] = sig
		}
		atts[i].Signature = bls.AggregateSignatures(sigs).Marshal()[:]
	}

	received, err = proposerServer.filterAttestationsForBlockInclusion(context.Background(), state, atts)
	if err != nil {
		t.Fatal(err)
	}
	if len(received) != 1 {
		t.Errorf("Did not filter correctly, wanted: 1, received: %d", len(received))
	}
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
	if err != nil {
		b.Fatal(err)
	}
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

	depositCache := depositcache.NewDepositCache()
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
		if err != nil {
			b.Fatal(err)
		}
		hashesByHeight[i] = blockhash
	}
	hashesByHeight[numOfVotes+1] = []byte("stub")

	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		b.Fatal(err)
	}

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
		if err != nil {
			b.Fatal(err)
		}
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
	if err != nil {
		t.Fatal(err)
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot(),
	}
	blkRoot, err := ssz.HashTreeRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	depositCache := depositcache.NewDepositCache()
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := ssz.HashTreeRoot(dp.Deposit.Data)
		if err != nil {
			t.Fatalf("Unable to determine hashed value of deposit %v", err)
		}

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
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != 0 {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: 0",
			len(deposits),
		)
	}
}

func TestDeleteAttsInPool_Aggregated(t *testing.T) {
	s := &Server{
		AttPool: attestations.NewPool(),
	}

	sig := bls.RandKey().Sign([]byte("foo")).Marshal()
	aggregatedAtts := []*ethpb.Attestation{{AggregationBits: bitfield.Bitlist{0b10101}, Signature: sig}, {AggregationBits: bitfield.Bitlist{0b11010}, Signature: sig}}
	unaggregatedAtts := []*ethpb.Attestation{{AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig}, {AggregationBits: bitfield.Bitlist{0b0001}, Signature: sig}}

	if err := s.AttPool.SaveAggregatedAttestations(aggregatedAtts); err != nil {
		t.Fatal(err)
	}
	if err := s.AttPool.SaveUnaggregatedAttestations(unaggregatedAtts); err != nil {
		t.Fatal(err)
	}

	aa, err := attaggregation.Aggregate(aggregatedAtts)
	if err != nil {
		t.Error(err)
	}
	if err := s.deleteAttsInPool(context.Background(), append(aa, unaggregatedAtts...)); err != nil {
		t.Fatal(err)
	}
	if len(s.AttPool.AggregatedAttestations()) != 0 {
		t.Error("Did not delete aggregated attestation")
	}
	if len(s.AttPool.UnaggregatedAttestations()) != 0 {
		t.Error("Did not delete unaggregated attestation")
	}
}
