package proposer

import (
	"context"
	"math/big"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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
	ctx := context.Background()

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(context.Background(), genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	numDeposits := params.BeaconConfig().MinGenesisActiveValidatorCount
	deposits, _, _ := testutil.SetupInitialDeposits(t, numDeposits)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
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

	proposerServer := &Server{
		BeaconDB:          db,
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
		BlockReceiver:     &mock.ChainService{},
		HeadFetcher:       &mock.ChainService{},
	}
	req := &ethpb.BeaconBlock{
		Slot:       5,
		ParentRoot: []byte("parent-hash"),
		Body:       &ethpb.BeaconBlockBody{},
	}
	if err := proposerServer.BeaconDB.SaveBlock(ctx, req); err != nil {
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

	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
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

	proposerServer := &Server{
		BeaconDB:          db,
		ChainStartFetcher: &mockPOW.POWChain{},
		Eth1InfoFetcher:   &mockPOW.POWChain{},
		Eth1BlockFetcher:  &mockPOW.POWChain{},
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
	domain := helpers.Domain(beaconState.Fork, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:], domain).Marshal()
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

	blk := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{Eth1Data: &ethpb.Eth1Data{}},
	}

	blkRoot, err := ssz.SigningRoot(blk)
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

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
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

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
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
