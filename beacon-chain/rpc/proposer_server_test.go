package rpc

import (
	"context"
	"crypto/rand"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func init() {
	// Use minimal config to reduce test setup time.
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
}

func TestProposeBlock_OK(t *testing.T) {
	helpers.ClearAllCaches()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	mockChain := &mockChainService{}
	ctx := context.Background()

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(context.Background(), genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	numDeposits := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _ := testutil.SetupInitialDeposits(t, numDeposits)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Could not instantiate genesis state: %v", err)
	}

	genesisRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, genesisRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}
	if err := db.SaveState(ctx, beaconState, genesisRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	proposerServer := &ProposerServer{
		chainService:    mockChain,
		beaconDB:        db,
		powChainService: &mockPOWChainService{},
	}
	req := &ethpb.BeaconBlock{
		Slot:       5,
		ParentRoot: []byte("parent-hash"),
	}
	if err := proposerServer.beaconDB.SaveBlock(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := proposerServer.ProposeBlock(context.Background(), req); err != nil {
		t.Errorf("Could not propose block correctly: %v", err)
	}
}

func TestComputeStateRoot_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	helpers.ClearAllCaches()

	mockChain := &mockChainService{}

	deposits, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Could not instantiate genesis state: %v", err)
	}

	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		t.Fatalf("Could not hash genesis state: %v", err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	if err := db.SaveBlock(ctx, genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	if err := db.SaveHeadBlockRoot(ctx, parentRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	if err := db.SaveState(ctx, beaconState, parentRoot); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	proposerServer := &ProposerServer{
		chainService:    mockChain,
		beaconDB:        db,
		powChainService: &mockPOWChainService{},
	}

	req := &ethpb.BeaconBlock{
		ParentRoot: parentRoot[:],
		Slot:       1,
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal:      nil,
			ProposerSlashings: nil,
			AttesterSlashings: nil,
			Eth1Data:          &ethpb.Eth1Data{},
		},
	}
	beaconState.Slot++
	randaoReveal, err := testutil.CreateRandaoReveal(beaconState, 0, privKeys)
	if err != nil {
		t.Error(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}
	beaconState.Slot--
	req.Body.RandaoReveal = randaoReveal[:]
	signingRoot, err := ssz.SigningRoot(req)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(beaconState)
	domain := helpers.Domain(beaconState, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:], domain).Marshal()
	req.Signature = blockSig[:]

	_, err = proposerServer.computeStateRoot(context.Background(), req)
	if err != nil {
		t.Error(err)
	}
}

func TestPendingAttestations_FiltersWithinInclusionDelay(t *testing.T) {
	helpers.ClearAllCaches()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	// This test breaks if it doesnt use mainnet config
	params.OverrideBeaconConfig(params.MainnetConfig())
	defer params.OverrideBeaconConfig(params.MinimalSpecConfig())
	ctx := context.Background()
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	crosslinks := make([]*ethpb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &ethpb.Crosslink{
			StartEpoch: 1,
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		}
	}

	stateSlot := uint64(100)
	beaconState := &pbp2p.BeaconState{
		Slot: stateSlot,
		Fork: &pbp2p.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		Validators:                  validators,
		CurrentCrosslinks:           crosslinks,
		PreviousCrosslinks:          crosslinks,
		StartShard:                  100,
		RandaoMixes:                 make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:            make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		FinalizedCheckpoint:         &ethpb.Checkpoint{},
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{},
	}

	encoded, err := ssz.HashTreeRoot(beaconState.PreviousCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard:      beaconState.Slot - params.BeaconConfig().MinAttestationInclusionDelay,
				DataRoot:   params.BeaconConfig().ZeroHash[:],
				ParentRoot: encoded[:]},
			Source: &ethpb.Checkpoint{},
			Target: &ethpb.Checkpoint{},
		},
		AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
		CustodyBits:     []byte{0x00, 0x00, 0x00, 0x00},
	}

	attestingIndices, err := helpers.AttestingIndices(beaconState, att.Data, att.AggregationBits)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(beaconState)
	domain := helpers.Domain(beaconState, currentEpoch, params.BeaconConfig().DomainAttestation)
	sigs := make([]*bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Error(err)
		}
		dataAndCustodyBit := &pbp2p.AttestationDataAndCustodyBit{
			Data:       att.Data,
			CustodyBit: false,
		}
		hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit)
		if err != nil {
			t.Error(err)
		}
		beaconState.Validators[indice].PublicKey = priv.PublicKey().Marshal()[:]
		sigs[i] = priv.Sign(hashTreeRoot[:], domain)
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	proposerServer := &ProposerServer{
		operationService: &mockOperationService{
			pendingAttestations: []*ethpb.Attestation{att},
		},
		chainService: &mockChainService{},
		beaconDB:     db,
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}
	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}

	atts, err := proposerServer.attestations(context.Background(), stateSlot)
	if err != nil {
		t.Fatalf("Unexpected error fetching pending attestations: %v", err)
	}
	if len(atts) == 0 {
		t.Error("Expected pending attestations list to be non-empty")
	}
}

func TestPendingAttestations_FiltersExpiredAttestations(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	// This test breaks if it doesnt use mainnet config
	params.OverrideBeaconConfig(params.MainnetConfig())
	defer params.OverrideBeaconConfig(params.MinimalSpecConfig())
	ctx := context.Background()

	// Edge case: current slot is at the end of an epoch. The pending attestation
	// for the next slot should come from currentSlot + 1.
	currentSlot := helpers.StartSlot(
		10,
	) - 1

	expectedEpoch := uint64(100)
	crosslink := &ethpb.Crosslink{StartEpoch: 9, DataRoot: params.BeaconConfig().ZeroHash[:]}
	encoded, err := ssz.HashTreeRoot(crosslink)
	if err != nil {
		t.Fatal(err)
	}

	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	beaconState := &pbp2p.BeaconState{
		Validators: validators,
		Slot:       currentSlot + params.BeaconConfig().MinAttestationInclusionDelay,
		Fork: &pbp2p.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: expectedEpoch,
		},
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: expectedEpoch,
		},
		CurrentCrosslinks: []*ethpb.Crosslink{{
			StartEpoch: 9,
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		}},
		RandaoMixes:       make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:  make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		StateRoots:        make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		BlockRoots:        make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		LatestBlockHeader: &ethpb.BeaconBlockHeader{StateRoot: []byte{}},
	}

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Target:    &ethpb.Checkpoint{Epoch: 10},
			Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
			Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
		},
		AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
	}
	attestingIndices, err := helpers.AttestingIndices(beaconState, att.Data, att.AggregationBits)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(beaconState, expectedEpoch, params.BeaconConfig().DomainAttestation)
	sigs := make([]*bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Error(err)
		}
		dataAndCustodyBit := &pbp2p.AttestationDataAndCustodyBit{
			Data:       att.Data,
			CustodyBit: false,
		}
		hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit)
		if err != nil {
			t.Error(err)
		}
		beaconState.Validators[indice].PublicKey = priv.PublicKey().Marshal()[:]
		sigs[i] = priv.Sign(hashTreeRoot[:], domain)
	}
	aggregateSig := bls.AggregateSignatures(sigs).Marshal()[:]
	att.Signature = aggregateSig

	att2 := proto.Clone(att).(*ethpb.Attestation)
	att3 := proto.Clone(att).(*ethpb.Attestation)

	opService := &mockOperationService{
		pendingAttestations: []*ethpb.Attestation{
			//Expired attestations
			{Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Epoch: 10},
				Source: &ethpb.Checkpoint{Epoch: expectedEpoch},

				Crosslink: &ethpb.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			{Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			{Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			{Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			{Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			// Non-expired attestation with incorrect justified epoch
			{Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch - 1},
				Crosslink: &ethpb.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			// Non-expired attestations with correct justified epoch
			att,
			att2,
			att3,
		},
	}
	expectedNumberOfAttestations := 3
	proposerServer := &ProposerServer{
		operationService: opService,
		chainService:     &mockChainService{},
		beaconDB:         db,
	}

	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}
	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}

	atts, err := proposerServer.attestations(context.Background(), currentSlot+params.BeaconConfig().MinAttestationInclusionDelay+1)
	if err != nil {
		t.Fatalf("Unexpected error fetching pending attestations: %v", err)
	}
	if len(atts) != expectedNumberOfAttestations {
		t.Errorf(
			"Expected pending attestations list length %d, but was %d",
			expectedNumberOfAttestations,
			len(atts),
		)
	}

	expectedAtts := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			Signature:       aggregateSig,
		},
		{
			Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			Signature:       aggregateSig,
		},
		{
			Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			Signature:       aggregateSig,
		},
	}
	if !reflect.DeepEqual(atts, expectedAtts) {
		t.Error("Did not receive expected attestations")
	}
}

func TestPendingDeposits_Eth1DataVoteOK(t *testing.T) {
	ctx := context.Background()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	newHeight := big.NewInt(height.Int64() + 11000)
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()):    []byte("0x0"),
			int(newHeight.Int64()): []byte("0x1"),
		},
	}

	var votes []*ethpb.Eth1Data

	vote := &ethpb.Eth1Data{
		BlockHash:    []byte("0x1"),
		DepositCount: 3,
	}
	for i := 0; i <= int(params.BeaconConfig().SlotsPerEth1VotingPeriod/2); i++ {
		votes = append(votes, vote)
	}

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 2,
		},
		Eth1DepositIndex: 2,
		Eth1DataVotes:    votes,
	}

	bs := &ProposerServer{
		beaconDB:        db,
		powChainService: p,
		chainService:    newMockChainService(),
	}

	blk := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{Eth1Data: &ethpb.Eth1Data{}},
	}

	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
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

	if proto.Equal(newState.Eth1Data, vote) {
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

	if !proto.Equal(newState.Eth1Data, vote) {
		t.Errorf("eth1data in the state not of the expected kind: Got %v but wanted %v", newState.Eth1Data, vote)
	}
}

func TestPendingDeposits_OutsideEth1FollowWindow(t *testing.T) {
	ctx := context.Background()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("0x0"),
		},
		Eth1DepositIndex: 2,
	}

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*depositcache.DepositContainer{
		{
			Index: 0,
			Block: big.NewInt(1000),
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Block: big.NewInt(1001),
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*depositcache.DepositContainer{
		{
			Index: 2,
			Block: big.NewInt(4000),
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("c"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 3,
			Block: big.NewInt(5000),
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

		if err := depositTrie.InsertIntoTrie(depositHash[:], int(dp.Index)); err != nil {
			t.Fatalf("Unable to insert deposit into trie %v", err)
		}

		depositCache.InsertDeposit(ctx, dp.Deposit, dp.Block, dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Block, dp.Index, depositTrie.Root())
	}

	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}

	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}

	bs := &ProposerServer{
		beaconDB:        db,
		powChainService: p,
		chainService:    newMockChainService(),
		depositCache:    depositCache,
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
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
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
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	newHeight := big.NewInt(height.Int64() + 11000)
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()):    []byte("0x0"),
			int(newHeight.Int64()): []byte("0x1"),
		},
	}

	var votes []*ethpb.Eth1Data

	vote := &ethpb.Eth1Data{
		BlockHash:    []byte("0x1"),
		DepositCount: 7,
	}
	for i := 0; i <= int(params.BeaconConfig().SlotsPerEth1VotingPeriod/2); i++ {
		votes = append(votes, vote)
	}

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 5,
		},
		Eth1DepositIndex: 1,
		Eth1DataVotes:    votes,
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}

	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*depositcache.DepositContainer{
		{
			Index: 0,
			Block: big.NewInt(1000),
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Block: big.NewInt(1010),
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*depositcache.DepositContainer{
		{
			Index: 2,
			Block: big.NewInt(5000),
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("c"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 3,
			Block: big.NewInt(6000),
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

		if err := depositTrie.InsertIntoTrie(depositHash[:], int(dp.Index)); err != nil {
			t.Fatalf("Unable to insert deposit into trie %v", err)
		}

		depositCache.InsertDeposit(ctx, dp.Deposit, dp.Block, dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, dp.Block, dp.Index, depositTrie.Root())
	}

	bs := &ProposerServer{
		beaconDB:        db,
		powChainService: p,
		chainService:    newMockChainService(),
		depositCache:    depositCache,
	}

	deposits, err := bs.deposits(ctx, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != 0 {
		t.Errorf("Received unexpected list of deposits: %+v, wanted: 0", len(deposits))
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
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
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 100,
		},
		Eth1DepositIndex: 10,
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}
	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}

	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*depositcache.DepositContainer{
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

	var recentDeposits []*depositcache.DepositContainer
	for i := 2; i < 16; i++ {
		recentDeposits = append(recentDeposits, &depositcache.DepositContainer{
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

		if err := depositTrie.InsertIntoTrie(depositHash[:], int(dp.Index)); err != nil {
			t.Fatalf("Unable to insert deposit into trie %v", err)
		}

		depositCache.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}

	bs := &ProposerServer{
		beaconDB:        db,
		powChainService: p,
		chainService:    newMockChainService(),
		depositCache:    depositCache,
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
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
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 100,
		},
		Eth1DepositIndex: 2,
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}
	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*depositcache.DepositContainer{
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

	var recentDeposits []*depositcache.DepositContainer
	for i := 2; i < 22; i++ {
		recentDeposits = append(recentDeposits, &depositcache.DepositContainer{
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

		if err := depositTrie.InsertIntoTrie(depositHash[:], int(dp.Index)); err != nil {
			t.Fatalf("Unable to insert deposit into trie %v", err)
		}

		depositCache.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}

	bs := &ProposerServer{
		beaconDB:        db,
		powChainService: p,
		chainService:    newMockChainService(),
		depositCache:    depositCache,
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
	deposits, err := bs.deposits(ctx, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != int(params.BeaconConfig().MaxDeposits) {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: %d",
			len(deposits),
			int(params.BeaconConfig().MaxDeposits),
		)
	}
}

func TestPendingDeposits_CantReturnMoreDepositCount(t *testing.T) {
	ctx := context.Background()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash:    []byte("0x0"),
			DepositCount: 5,
		},
		Eth1DepositIndex: 2,
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}
	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*depositcache.DepositContainer{
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

	var recentDeposits []*depositcache.DepositContainer
	for i := 2; i < 22; i++ {
		recentDeposits = append(recentDeposits, &depositcache.DepositContainer{
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

		if err := depositTrie.InsertIntoTrie(depositHash[:], int(dp.Index)); err != nil {
			t.Fatalf("Unable to insert deposit into trie %v", err)
		}

		depositCache.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}

	bs := &ProposerServer{
		beaconDB:        db,
		powChainService: p,
		chainService:    newMockChainService(),
		depositCache:    depositCache,
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
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
	ctx := context.Background()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	proposerServer := &ProposerServer{
		beaconDB: db,
		powChainService: &faultyPOWChainService{
			hashesByHeight: make(map[int][]byte),
		},
	}
	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte{'a'},
		},
		Eth1DataVotes: []*ethpb.Eth1Data{},
	}
	if err := proposerServer.beaconDB.SaveState(ctx, beaconState, [32]byte{}); err != nil {
		t.Fatal(err)
	}
	want := "could not fetch ETH1_FOLLOW_DISTANCE ancestor"
	if _, err := proposerServer.eth1Data(context.Background(), beaconState.Slot+1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error %v, received %v", want, err)
	}
}

func TestDefaultEth1Data_NoBlockExists(t *testing.T) {
	ctx := context.Background()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	deps := []*depositcache.DepositContainer{
		{
			Index: 0,
			Block: big.NewInt(1000),
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Block: big.NewInt(1200),
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	depositCache := depositcache.NewDepositCache()
	for _, dp := range deps {
		depositCache.InsertDeposit(context.Background(), dp.Deposit, dp.Block, dp.Index, depositTrie.Root())
	}

	powChainService := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			0:   []byte("hash0"),
			476: []byte("hash1024"),
		},
	}
	proposerServer := &ProposerServer{
		beaconDB:        db,
		powChainService: powChainService,
		depositCache:    depositCache,
	}

	defEth1Data := &ethpb.Eth1Data{
		DepositCount: 10,
		BlockHash:    []byte{'t', 'e', 's', 't'},
		DepositRoot:  []byte{'r', 'o', 'o', 't'},
	}

	powChainService.eth1Data = defEth1Data

	result, err := proposerServer.defaultEth1DataResponse(ctx, big.NewInt(1500))
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(result, defEth1Data) {
		t.Errorf("Did not receive default eth1data. Wanted %v but Got %v", defEth1Data, result)
	}
}

// TODO(2312): Add more tests for edge cases and better coverage.
func TestEth1Data(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	slot := uint64(10000)

	ps := &ProposerServer{
		powChainService: &mockPOWChainService{
			blockNumberByHeight: map[uint64]*big.Int{
				60000: big.NewInt(4096),
			},
			hashesByHeight: map[int][]byte{
				3072: []byte("3072"),
			},
			eth1Data: &ethpb.Eth1Data{
				DepositCount: 55,
			},
		},
		beaconDB:     db,
		depositCache: depositcache.NewDepositCache(),
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

func Benchmark_Eth1Data(b *testing.B) {
	ctx := context.Background()
	db := dbutil.SetupDB(b)
	defer dbutil.TeardownDB(b, db)

	hashesByHeight := make(map[int][]byte)

	beaconState := &pbp2p.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{},
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("stub"),
		},
	}
	var mockSig [96]byte
	var mockCreds [32]byte
	deposits := []*depositcache.DepositContainer{
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
		depositCache.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, root)
	}
	numOfVotes := 1000
	for i := 0; i < numOfVotes; i++ {
		blockhash := []byte{'b', 'l', 'o', 'c', 'k', byte(i)}
		deposit := []byte{'d', 'e', 'p', 'o', 's', 'i', 't', byte(i)}
		beaconState.Eth1DataVotes = append(beaconState.Eth1DataVotes, &ethpb.Eth1Data{
			BlockHash:   blockhash,
			DepositRoot: deposit,
		})
		hashesByHeight[i] = blockhash
	}
	hashesByHeight[numOfVotes+1] = []byte("stub")

	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}
	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		b.Fatal(err)
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		b.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		b.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		b.Fatal(err)
	}
	currentHeight := params.BeaconConfig().Eth1FollowDistance + 5
	proposerServer := &ProposerServer{
		beaconDB: db,
		powChainService: &mockPOWChainService{
			latestBlockNumber: big.NewInt(int64(currentHeight)),
			hashesByHeight:    hashesByHeight,
		},
		depositCache: depositCache,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proposerServer.eth1Data(context.Background(), beaconState.Slot+1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDeposits_ReturnsEmptyList_IfLatestEth1DataEqGenesisEth1Block(t *testing.T) {
	ctx := context.Background()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
		genesisEth1Block: height,
	}

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("0x0"),
		},
		Eth1DepositIndex: 2,
	}
	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}
	blkRoot, err := ssz.SigningRoot(blk)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, beaconState, blkRoot); err != nil {
		t.Fatal(err)
	}
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*depositcache.DepositContainer{
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

	var recentDeposits []*depositcache.DepositContainer
	for i := 2; i < 22; i++ {
		recentDeposits = append(recentDeposits, &depositcache.DepositContainer{
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

		if err := depositTrie.InsertIntoTrie(depositHash[:], int(dp.Index)); err != nil {
			t.Fatalf("Unable to insert deposit into trie %v", err)
		}

		depositCache.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		depositCache.InsertPendingDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}

	bs := &ProposerServer{
		beaconDB:        db,
		powChainService: p,
		chainService:    newMockChainService(),
		depositCache:    depositCache,
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
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
