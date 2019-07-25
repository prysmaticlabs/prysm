package rpc

import (
	"context"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func init() {
	// Use minimal config to reduce test setup time.
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
}

func TestProposeBlock_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	mockChain := &mockChainService{}
	ctx := context.Background()

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	numDeposits := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _ := testutil.SetupInitialDeposits(t, numDeposits)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatalf("Could not instantiate genesis state: %v", err)
	}

	if err := db.UpdateChainHead(ctx, genesis, beaconState); err != nil {
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
	if err := proposerServer.beaconDB.SaveBlock(req); err != nil {
		t.Fatal(err)
	}
	if _, err := proposerServer.ProposeBlock(context.Background(), req); err != nil {
		t.Errorf("Could not propose block correctly: %v", err)
	}
}

func TestComputeStateRoot_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
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
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	if err := db.UpdateChainHead(ctx, genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	parentRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
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
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}
	beaconState.Slot--
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

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
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
		Slot:                        stateSlot,
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

	proposerServer := &ProposerServer{
		operationService: &mockOperationService{
			pendingAttestations: []*ethpb.Attestation{
				{Data: &ethpb.AttestationData{
					Crosslink: &ethpb.Crosslink{
						Shard:      beaconState.Slot - params.BeaconConfig().MinAttestationInclusionDelay,
						DataRoot:   params.BeaconConfig().ZeroHash[:],
						ParentRoot: encoded[:]},
					Source: &ethpb.Checkpoint{},
					Target: &ethpb.Checkpoint{},
				},
					AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
					CustodyBits:     []byte{0x00, 0x00, 0x00, 0x00},
				},
			},
		},
		chainService: &mockChainService{},
		beaconDB:     db,
	}
	if err := db.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}

	if err := db.SaveBlock(blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}

	if err := db.UpdateChainHead(ctx, blk, beaconState); err != nil {
		t.Fatalf("couldnt update chainhead: %v", err)
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
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
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
			{Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			}, AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01}},
			{Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			}, AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01}},
			{Data: &ethpb.AttestationData{
				Target:    &ethpb.Checkpoint{Epoch: 10},
				Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
				Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			}, AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01}},
		},
	}
	expectedNumberOfAttestations := 3
	proposerServer := &ProposerServer{
		operationService: opService,
		chainService:     &mockChainService{},
		beaconDB:         db,
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

	if err := db.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	blk := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}

	if err := db.SaveBlock(blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}

	if err := db.UpdateChainHead(ctx, blk, beaconState); err != nil {
		t.Fatalf("couldnt update chainhead: %v", err)
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
		{Data: &ethpb.AttestationData{
			Target:    &ethpb.Checkpoint{Epoch: 10},
			Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
			Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
		}, AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01}},
		{Data: &ethpb.AttestationData{
			Target:    &ethpb.Checkpoint{Epoch: 10},
			Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
			Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
		}, AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01}},
		{Data: &ethpb.AttestationData{
			Target:    &ethpb.Checkpoint{Epoch: 10},
			Source:    &ethpb.Checkpoint{Epoch: expectedEpoch},
			Crosslink: &ethpb.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
		}, AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01}},
	}
	if !reflect.DeepEqual(atts, expectedAtts) {
		t.Error("Did not receive expected attestations")
	}
}

func TestPendingDeposits_UnknownBlockNum(t *testing.T) {
	p := &mockPOWChainService{
		latestBlockNumber: nil,
	}
	ps := ProposerServer{powChainService: p}

	_, err := ps.deposits(context.Background(), &ethpb.Eth1Data{})
	if err.Error() != "latest PoW block number is unknown" {
		t.Errorf("Received unexpected error: %v", err)
	}
}

func TestPendingDeposits_OutsideEth1FollowWindow(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}
	d := internal.SetupDB(t)

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("0x0"),
		},
		Eth1DepositIndex: 2,
	}
	if err := d.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*db.DepositContainer{
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

	recentDeposits := []*db.DepositContainer{
		{
			Index: 2,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("c"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 3,
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("d"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := hashutil.DepositHash(dp.Deposit.Data)
		if err != nil {
			t.Fatalf("Unable to determine hashed value of deposit %v", err)
		}

		if err := depositTrie.InsertIntoTrie(depositHash[:], int(dp.Index)); err != nil {
			t.Fatalf("Unable to insert deposit into trie %v", err)
		}

		d.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		d.InsertPendingDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}

	bs := &ProposerServer{
		beaconDB:        d,
		powChainService: p,
		chainService:    newMockChainService(),
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
	deposits, err = bs.deposits(ctx, &ethpb.Eth1Data{})
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
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}
	d := internal.SetupDB(t)

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("0x0"),
		},
		Eth1DepositIndex: 10,
	}
	if err := d.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*db.DepositContainer{
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

	var recentDeposits []*db.DepositContainer
	for i := 2; i < 16; i++ {
		recentDeposits = append(recentDeposits, &db.DepositContainer{
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
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := hashutil.DepositHash(dp.Deposit.Data)
		if err != nil {
			t.Fatalf("Unable to determine hashed value of deposit %v", err)
		}

		if err := depositTrie.InsertIntoTrie(depositHash[:], int(dp.Index)); err != nil {
			t.Fatalf("Unable to insert deposit into trie %v", err)
		}

		d.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		d.InsertPendingDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}

	bs := &ProposerServer{
		beaconDB:        d,
		powChainService: p,
		chainService:    newMockChainService(),
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

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}
	d := internal.SetupDB(t)

	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("0x0"),
		},
		Eth1DepositIndex: 2,
	}
	if err := d.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*db.DepositContainer{
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

	var recentDeposits []*db.DepositContainer
	for i := 2; i < 22; i++ {
		recentDeposits = append(recentDeposits, &db.DepositContainer{
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
	for _, dp := range append(readyDeposits, recentDeposits...) {
		depositHash, err := hashutil.DepositHash(dp.Deposit.Data)
		if err != nil {
			t.Fatalf("Unable to determine hashed value of deposit %v", err)
		}

		if err := depositTrie.InsertIntoTrie(depositHash[:], int(dp.Index)); err != nil {
			t.Fatalf("Unable to insert deposit into trie %v", err)
		}

		d.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}
	for _, dp := range recentDeposits {
		d.InsertPendingDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, depositTrie.Root())
	}

	bs := &ProposerServer{
		beaconDB:        d,
		powChainService: p,
		chainService:    newMockChainService(),
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

func TestEth1Data_EmptyVotesFetchBlockHashFailure(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

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
	if err := proposerServer.beaconDB.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	want := "could not fetch ETH1_FOLLOW_DISTANCE ancestor"
	if _, err := proposerServer.eth1Data(context.Background(), beaconState.Slot+1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error %v, received %v", want, err)
	}
}

func TestDefaultEth1Data_NoBlockExists(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	deps := []*db.DepositContainer{
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
	for _, dp := range deps {
		beaconDB.InsertDeposit(context.Background(), dp.Deposit, dp.Block, dp.Index, depositTrie.Root())
	}

	powChainService := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			0:   []byte("hash0"),
			476: []byte("hash1024"),
		},
	}
	proposerServer := &ProposerServer{
		beaconDB:        beaconDB,
		powChainService: powChainService,
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
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

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
		beaconDB: beaconDB,
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
	beaconDB := internal.SetupDB(b)
	defer internal.TeardownDB(b, beaconDB)
	ctx := context.Background()

	hashesByHeight := make(map[int][]byte)

	beaconState := &pbp2p.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{},
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte("stub"),
		},
	}
	var mockSig [96]byte
	var mockCreds [32]byte
	deposits := []*db.DepositContainer{
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

	for i, dp := range deposits {
		var root [32]byte
		copy(root[:], []byte{'d', 'e', 'p', 'o', 's', 'i', 't', byte(i)})
		beaconDB.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, root)
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

	if err := beaconDB.SaveState(ctx, beaconState); err != nil {
		b.Fatal(err)
	}
	currentHeight := params.BeaconConfig().Eth1FollowDistance + 5
	proposerServer := &ProposerServer{
		beaconDB: beaconDB,
		powChainService: &mockPOWChainService{
			latestBlockNumber: big.NewInt(int64(currentHeight)),
			hashesByHeight:    hashesByHeight,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proposerServer.eth1Data(context.Background(), beaconState.Slot+1)
		if err != nil {
			b.Fatal(err)
		}
	}
}
