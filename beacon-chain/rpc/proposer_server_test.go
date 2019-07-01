package rpc

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableComputeStateRoot: true,
	})
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

	numDeposits := params.BeaconConfig().DepositsForChainStart
	deposits, _ := testutil.SetupInitialDeposits(t, numDeposits, false)
	beaconState, err := state.GenesisBeaconState(deposits, 0, nil)
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
	req := &pbp2p.BeaconBlock{
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

	mockChain := &mockChainService{}

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	deposits, _ := testutil.SetupInitialDeposits(t, params.BeaconConfig().DepositsForChainStart, false)
	beaconState, err := state.GenesisBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Could not instantiate genesis state: %v", err)
	}
	beaconState.StateRoots = make([][]byte, params.BeaconConfig().HistoricalRootsLimit)
	beaconState.LatestBlockHeader = &pbp2p.BeaconBlockHeader{
		StateRoot: []byte{},
	}
	beaconState.Slot = 10

	if err := db.UpdateChainHead(ctx, genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	proposerServer := &ProposerServer{
		chainService:    mockChain,
		beaconDB:        db,
		powChainService: &mockPOWChainService{},
	}

	req := &pbp2p.BeaconBlock{
		ParentRoot: nil,
		Slot:       11,
		Body: &pbp2p.BeaconBlockBody{
			RandaoReveal:      nil,
			ProposerSlashings: nil,
			AttesterSlashings: nil,
			Eth1Data:          &pbp2p.Eth1Data{},
		},
	}

	_, _ = proposerServer.computeStateRoot(context.Background(), req)
}

func TestPendingAttestations_FiltersWithinInclusionDelay(t *testing.T) {
	helpers.ClearAllCaches()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	validators := make([]*pbp2p.Validator, params.BeaconConfig().DepositsForChainStart/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pbp2p.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	crosslinks := make([]*pbp2p.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &pbp2p.Crosslink{
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
		FinalizedCheckpoint:         &pbp2p.Checkpoint{},
		PreviousJustifiedCheckpoint: &pbp2p.Checkpoint{},
		CurrentJustifiedCheckpoint:  &pbp2p.Checkpoint{},
	}

	encoded, err := ssz.HashTreeRoot(beaconState.PreviousCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}

	proposerServer := &ProposerServer{
		operationService: &mockOperationService{
			pendingAttestations: []*pbp2p.Attestation{
				{Data: &pbp2p.AttestationData{
					Crosslink: &pbp2p.Crosslink{
						Shard:      beaconState.Slot - params.BeaconConfig().MinAttestationInclusionDelay,
						DataRoot:   params.BeaconConfig().ZeroHash[:],
						ParentRoot: encoded[:]},
				},
					AggregationBits: []byte{0xC0, 0xC0, 0xC0, 0xC0},
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

	blk := &pbp2p.BeaconBlock{
		Slot: beaconState.Slot,
	}

	if err := db.SaveBlock(blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}

	if err := db.UpdateChainHead(ctx, blk, beaconState); err != nil {
		t.Fatalf("couldnt update chainhead: %v", err)
	}

	atts, err := proposerServer.attestations(context.Background())
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
	ctx := context.Background()

	// Edge case: current slot is at the end of an epoch. The pending attestation
	// for the next slot should come from currentSlot + 1.
	currentSlot := helpers.StartSlot(
		10,
	) - 1

	expectedEpoch := uint64(100)
	crosslink := &pbp2p.Crosslink{StartEpoch: 9, DataRoot: params.BeaconConfig().ZeroHash[:]}
	encoded, err := ssz.HashTreeRoot(crosslink)
	if err != nil {
		t.Fatal(err)
	}

	opService := &mockOperationService{
		pendingAttestations: []*pbp2p.Attestation{
			//Expired attestations
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			// Non-expired attestation with incorrect justified epoch
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch - 1,
				Crosslink:   &pbp2p.Crosslink{DataRoot: params.BeaconConfig().ZeroHash[:]},
			}},
			// Non-expired attestations with correct justified epoch
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			}, AggregationBits: []byte{0xC0, 0xC0, 0xC0, 0xC0}, CustodyBits: []byte{0x00, 0x00, 0x00, 0x00},
			},
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			}, AggregationBits: []byte{0xC0, 0xC0, 0xC0, 0xC0}, CustodyBits: []byte{0x00, 0x00, 0x00, 0x00},
			},
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			}, AggregationBits: []byte{0xC0, 0xC0, 0xC0, 0xC0}, CustodyBits: []byte{0x00, 0x00, 0x00, 0x00},
			},
		},
	}
	expectedNumberOfAttestations := 3
	proposerServer := &ProposerServer{
		operationService: opService,
		chainService:     &mockChainService{},
		beaconDB:         db,
	}

	validators := make([]*pbp2p.Validator, params.BeaconConfig().DepositsForChainStart/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pbp2p.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	beaconState := &pbp2p.BeaconState{
		Validators: validators,
		Slot:       currentSlot + params.BeaconConfig().MinAttestationInclusionDelay,
		CurrentJustifiedCheckpoint: &pbp2p.Checkpoint{
			Epoch: expectedEpoch,
		},
		PreviousJustifiedCheckpoint: &pbp2p.Checkpoint{
			Epoch: expectedEpoch,
		},
		CurrentCrosslinks: []*pbp2p.Crosslink{{
			StartEpoch: 9,
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		}},
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	if err := db.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	blk := &pbp2p.BeaconBlock{
		Slot: beaconState.Slot,
	}

	if err := db.SaveBlock(blk); err != nil {
		t.Fatalf("failed to save block %v", err)
	}

	if err := db.UpdateChainHead(ctx, blk, beaconState); err != nil {
		t.Fatalf("couldnt update chainhead: %v", err)
	}

	atts, err := proposerServer.attestations(context.Background())
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

	expectedAtts := []*pbp2p.Attestation{
		{Data: &pbp2p.AttestationData{
			TargetEpoch: 10,
			SourceEpoch: expectedEpoch,
			Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
		}, AggregationBits: []byte{0xC0, 0xC0, 0xC0, 0xC0}, CustodyBits: []byte{0x00, 0x00, 0x00, 0x00}},
		{Data: &pbp2p.AttestationData{
			TargetEpoch: 10,
			SourceEpoch: expectedEpoch,
			Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
		}, AggregationBits: []byte{0xC0, 0xC0, 0xC0, 0xC0}, CustodyBits: []byte{0x00, 0x00, 0x00, 0x00}},
		{Data: &pbp2p.AttestationData{
			TargetEpoch: 10,
			SourceEpoch: expectedEpoch,
			Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
		}, AggregationBits: []byte{0xC0, 0xC0, 0xC0, 0xC0}, CustodyBits: []byte{0x00, 0x00, 0x00, 0x00}},
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

	_, err := ps.deposits(context.Background())
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
		Eth1Data: &pbp2p.Eth1Data{
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
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	recentDeposits := []*db.DepositContainer{
		{
			Index: 2,
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte("c"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 3,
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte("d"),
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

	deposits, err := bs.deposits(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != 0 {
		t.Errorf("Received unexpected list of deposits: %+v, wanted: 0", len(deposits))
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
	deposits, err = bs.deposits(ctx)
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
		Eth1Data: &pbp2p.Eth1Data{
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
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*db.DepositContainer
	for i := 2; i < 16; i++ {
		recentDeposits = append(recentDeposits, &db.DepositContainer{
			Index: i,
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte{byte(i)},
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
	deposits, err := bs.deposits(ctx)
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
		Eth1Data: &pbp2p.Eth1Data{
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
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte("a"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
		{
			Index: 1,
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte("b"),
					Signature:             mockSig[:],
					WithdrawalCredentials: mockCreds[:],
				}},
		},
	}

	var recentDeposits []*db.DepositContainer
	for i := 2; i < 22; i++ {
		recentDeposits = append(recentDeposits, &db.DepositContainer{
			Index: i,
			Deposit: &pbp2p.Deposit{
				Data: &pbp2p.DepositData{
					Pubkey:                []byte{byte(i)},
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
	deposits, err := bs.deposits(ctx)
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
