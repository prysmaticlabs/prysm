package rpc

import (
	"bytes"
	"context"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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

	deposits := make([]*pbp2p.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		depositData := &pbp2p.DepositData{
			Pubkey: []byte(strconv.Itoa(i)),
			Amount: params.BeaconConfig().MaxDepositAmount,
		}
		deposits[i] = &pbp2p.Deposit{
			Data: depositData,
		}
	}

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

	deposits := make([]*pbp2p.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		depositData := &pbp2p.DepositData{
			Pubkey: []byte(strconv.Itoa(i)),
			Amount: params.BeaconConfig().MaxDepositAmount,
		}
		deposits[i] = &pbp2p.Deposit{
			Data: depositData,
		}
		deposits[i] = &pbp2p.Deposit{
			Data: depositData,
		}
	}

	beaconState, err := state.GenesisBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Could not instantiate genesis state: %v", err)
	}
	beaconState.LatestStateRoots = make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
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
		Slot:                   stateSlot,
		ValidatorRegistry:      validators,
		CurrentCrosslinks:      crosslinks,
		PreviousCrosslinks:     crosslinks,
		LatestStartShard:       100,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	encoded, err := ssz.TreeHash(beaconState.PreviousCrosslinks[0])
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
					AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0},
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
	encoded, err := ssz.TreeHash(crosslink)
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
			}, AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0}},
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			}, AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0}},
			{Data: &pbp2p.AttestationData{
				TargetEpoch: 10,
				SourceEpoch: expectedEpoch,
				Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
			}, AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0}},
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
		ValidatorRegistry:      validators,
		Slot:                   currentSlot + params.BeaconConfig().MinAttestationInclusionDelay,
		CurrentJustifiedEpoch:  expectedEpoch,
		PreviousJustifiedEpoch: expectedEpoch,
		CurrentCrosslinks: []*pbp2p.Crosslink{{
			StartEpoch: 9,
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		}},
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
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
		}, AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0}},
		{Data: &pbp2p.AttestationData{
			TargetEpoch: 10,
			SourceEpoch: expectedEpoch,
			Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
		}, AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0}},
		{Data: &pbp2p.AttestationData{
			TargetEpoch: 10,
			SourceEpoch: expectedEpoch,
			Crosslink:   &pbp2p.Crosslink{EndEpoch: 10, DataRoot: params.BeaconConfig().ZeroHash[:], ParentRoot: encoded[:]},
		}, AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0}},
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
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockRoot: []byte("0x0"),
		},
		DepositIndex: 2,
	}
	if err := d.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	var mockSig [96]byte
	var mockCreds [32]byte

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*pbp2p.Deposit{
		{
			Index: 0,
			Data: &pbp2p.DepositData{
				Pubkey:                []byte("a"),
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		},
		{
			Index: 1,
			Data: &pbp2p.DepositData{
				Pubkey:                []byte("b"),
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		},
	}

	recentDeposits := []*pbp2p.Deposit{
		{
			Index: 2,
			Data: &pbp2p.DepositData{
				Pubkey:                []byte("c"),
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		},
		{
			Index: 3,
			Data: &pbp2p.DepositData{
				Pubkey:                []byte("d"),
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		},
	}
	for _, dp := range append(readyDeposits, recentDeposits...) {
		d.InsertDeposit(ctx, dp, big.NewInt(int64(dp.Index)))
	}
	for _, dp := range recentDeposits {
		d.InsertPendingDeposit(ctx, dp, big.NewInt(int64(dp.Index)))
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

func Benchmark_Eth1Data(b *testing.B) {
	db := internal.SetupDB(b)
	defer internal.TeardownDB(b, db)
	ctx := context.Background()

	hashesByHeight := make(map[int][]byte)

	beaconState := &pbp2p.BeaconState{
		Eth1DataVotes: []*pbp2p.Eth1Data{},
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockRoot: []byte("stub"),
		},
	}
	numOfVotes := 1000
	for i := 0; i < numOfVotes; i++ {
		blockhash := []byte{'b', 'l', 'o', 'c', 'k', byte(i)}
		deposit := []byte{'d', 'e', 'p', 'o', 's', 'i', 't', byte(i)}
		beaconState.Eth1DataVotes = append(beaconState.Eth1DataVotes, &pbp2p.Eth1Data{
			BlockRoot:   blockhash,
			DepositRoot: deposit,
		})
		hashesByHeight[i] = blockhash
	}
	hashesByHeight[numOfVotes+1] = []byte("stub")

	if err := db.SaveState(ctx, beaconState); err != nil {
		b.Fatal(err)
	}
	currentHeight := params.BeaconConfig().Eth1FollowDistance + 5
	proposerServer := &ProposerServer{
		beaconDB: db,
		powChainService: &mockPOWChainService{
			latestBlockNumber: big.NewInt(int64(currentHeight)),
			hashesByHeight:    hashesByHeight,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proposerServer.eth1Data(context.Background())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestPendingDeposits_CantReturnBelowStateDepositIndex(t *testing.T) {
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
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockRoot: []byte("0x0"),
		},
		DepositIndex: 10,
	}
	if err := d.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*pbp2p.Deposit{
		{
			Index: 0,
			Data: &pbp2p.DepositData{
				Pubkey:                []byte("a"),
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		},
		{
			Index: 1,
			Data: &pbp2p.DepositData{
				Pubkey:                []byte("b"),
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		},
	}

	var recentDeposits []*pbp2p.Deposit
	for i := 2; i < 16; i++ {
		recentDeposits = append(recentDeposits, &pbp2p.Deposit{
			Index: uint64(i),
			Data: &pbp2p.DepositData{
				Pubkey:                []byte{byte(i)},
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		})
	}

	for _, dp := range append(readyDeposits, recentDeposits...) {
		d.InsertDeposit(ctx, dp, big.NewInt(int64(dp.Index)))
	}
	for _, dp := range recentDeposits {
		d.InsertPendingDeposit(ctx, dp, big.NewInt(int64(dp.Index)))
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
	if deposits[0].Index != beaconState.DepositIndex {
		t.Errorf(
			"Received unexpected merkle index: %d, wanted: %d",
			deposits[0].Index,
			beaconState.DepositIndex,
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
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockRoot: []byte("0x0"),
		},
		DepositIndex: 2,
	}
	if err := d.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	var mockSig [96]byte
	var mockCreds [32]byte

	readyDeposits := []*pbp2p.Deposit{
		{
			Index: 0,
			Data: &pbp2p.DepositData{
				Pubkey:                []byte("a"),
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		},
		{
			Index: 1,
			Data: &pbp2p.DepositData{
				Pubkey:                []byte("b"),
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		},
	}

	var recentDeposits []*pbp2p.Deposit
	for i := 2; i < 22; i++ {
		recentDeposits = append(recentDeposits, &pbp2p.Deposit{
			Index: uint64(i),
			Data: &pbp2p.DepositData{
				Pubkey:                []byte{byte(i)},
				Signature:             mockSig[:],
				WithdrawalCredentials: mockCreds[:],
			},
		})
	}

	for _, dp := range append(readyDeposits, recentDeposits...) {
		d.InsertDeposit(ctx, dp, big.NewInt(int64(dp.Index)))
	}
	for _, dp := range recentDeposits {
		d.InsertPendingDeposit(ctx, dp, big.NewInt(int64(dp.Index)))
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

func TestEth1Data_EmptyVotesFetchBlockHashFailure(t *testing.T) {
	t.Skip()
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
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockRoot: []byte{'a'},
		},
		Eth1DataVotes: []*pbp2p.Eth1Data{},
	}
	if err := proposerServer.beaconDB.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	want := "could not fetch ETH1_FOLLOW_DISTANCE ancestor"
	if _, err := proposerServer.eth1Data(context.Background()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error %v, received %v", want, err)
	}
}

func TestEth1Data_EmptyVotesOk(t *testing.T) {
	t.Skip()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	deps := []*pbp2p.Deposit{
		{Index: 0, Data: &pbp2p.DepositData{
			Pubkey: []byte("a"),
		}},
		{Index: 1, Data: &pbp2p.DepositData{
			Pubkey: []byte("b"),
		}},
	}
	depsData := [][]byte{}
	for _, dp := range deps {
		db.InsertDeposit(context.Background(), dp, big.NewInt(0))
		depHash, err := hashutil.DepositHash(dp.Data)
		if err != nil {
			t.Errorf("Could not hash deposit")
		}
		depsData = append(depsData, depHash[:])
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(depsData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}
	depositRoot := depositTrie.Root()
	beaconState := &pbp2p.BeaconState{
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockRoot:   []byte("hash0"),
			DepositRoot: depositRoot[:],
		},
		Eth1DataVotes: []*pbp2p.Eth1Data{},
	}

	powChainService := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			0: []byte("hash0"),
			1: beaconState.LatestEth1Data.BlockRoot,
		},
	}
	proposerServer := &ProposerServer{
		beaconDB:        db,
		powChainService: powChainService,
	}

	if err := proposerServer.beaconDB.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	eth1data, err := proposerServer.eth1Data(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// If the data vote objects are empty, the deposit root should be the one corresponding
	// to the deposit contract in the powchain service, fetched using powChainService.DepositRoot()
	if !bytes.Equal(eth1data.DepositRoot, depositRoot[:]) {
		t.Errorf(
			"Expected deposit roots to match, received %#x == %#x",
			eth1data.DepositRoot,
			depositRoot,
		)
	}
}

func TestEth1Data_NonEmptyVotesSelectsBestVote(t *testing.T) {
	t.Skip()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	eth1DataVotes := []*pbp2p.Eth1Data{}
	beaconState := &pbp2p.BeaconState{
		Eth1DataVotes: eth1DataVotes,
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockRoot: []byte("stub"),
		},
	}
	if err := db.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	currentHeight := params.BeaconConfig().Eth1FollowDistance + 5
	proposerServer := &ProposerServer{
		beaconDB: db,
		powChainService: &mockPOWChainService{
			latestBlockNumber: big.NewInt(int64(currentHeight)),
			hashesByHeight: map[int][]byte{
				0: beaconState.LatestEth1Data.BlockRoot,
				1: beaconState.Eth1DataVotes[0].BlockRoot,
				2: beaconState.Eth1DataVotes[1].BlockRoot,
				3: beaconState.Eth1DataVotes[3].BlockRoot,
				// We will give the hash at index 2 in the beacon state's latest eth1 votes
				// priority in being selected as the best vote by giving it the highest block number.
				4: beaconState.Eth1DataVotes[2].BlockRoot,
			},
		},
	}
	eth1data, err := proposerServer.eth1Data(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Vote at index 2 should have won the best vote selection mechanism as it had the highest block number
	// despite being tied at vote count with the vote at index 3.
	if !bytes.Equal(eth1data.BlockRoot, beaconState.Eth1DataVotes[2].BlockRoot) {
		t.Errorf(
			"Expected block hashes to match, received %#x == %#x",
			eth1data.BlockRoot,
			beaconState.Eth1DataVotes[2].BlockRoot,
		)
	}
	if !bytes.Equal(eth1data.DepositRoot, beaconState.Eth1DataVotes[2].DepositRoot) {
		t.Errorf(
			"Expected deposit roots to match, received %#x == %#x",
			eth1data.DepositRoot,
			beaconState.Eth1DataVotes[2].DepositRoot,
		)
	}
}
