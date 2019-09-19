package rpc

import (
	"context"
	"math/big"
	"strings"
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
	helpers.ClearAllCaches()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
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
		beaconDB:          db,
		chainStartFetcher: &mockPOW.POWChain{},
		eth1InfoFetcher:   &mockPOW.POWChain{},
		eth1BlockFetcher:  &mockPOW.POWChain{},
		blockReceiver:     &mock.ChainService{},
		headFetcher:       &mock.ChainService{},
	}
	req := &ethpb.BeaconBlock{
		Slot:       5,
		ParentRoot: []byte("parent-hash"),
		Body:       &ethpb.BeaconBlockBody{},
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
		beaconDB:          db,
		chainStartFetcher: &mockPOW.POWChain{},
		eth1InfoFetcher:   &mockPOW.POWChain{},
		eth1BlockFetcher:  &mockPOW.POWChain{},
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

	bs := &ProposerServer{
		chainStartFetcher: p,
		eth1InfoFetcher:   p,
		eth1BlockFetcher:  p,
		blockReceiver:     &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		headFetcher:       &mock.ChainService{State: beaconState, Root: blkRoot[:]},
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

	bs := &ProposerServer{
		chainStartFetcher:      p,
		eth1InfoFetcher:        p,
		eth1BlockFetcher:       p,
		depositFetcher:         depositCache,
		pendingDepositsFetcher: depositCache,
		blockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		headFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
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

	bs := &ProposerServer{
		chainStartFetcher:      p,
		eth1InfoFetcher:        p,
		eth1BlockFetcher:       p,
		depositFetcher:         depositCache,
		pendingDepositsFetcher: depositCache,
		blockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		headFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
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

	bs := &ProposerServer{
		chainStartFetcher:      p,
		eth1InfoFetcher:        p,
		eth1BlockFetcher:       p,
		depositFetcher:         depositCache,
		pendingDepositsFetcher: depositCache,
		blockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		headFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
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

	bs := &ProposerServer{
		chainStartFetcher:      p,
		eth1InfoFetcher:        p,
		eth1BlockFetcher:       p,
		depositFetcher:         depositCache,
		pendingDepositsFetcher: depositCache,
		blockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		headFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
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

	bs := &ProposerServer{
		blockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		headFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		chainStartFetcher:      p,
		eth1InfoFetcher:        p,
		eth1BlockFetcher:       p,
		depositFetcher:         depositCache,
		pendingDepositsFetcher: depositCache,
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
	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte{'a'},
		},
		Eth1DataVotes: []*ethpb.Eth1Data{},
	}
	p := &mockPOW.FaultyMockPOWChain{
		HashesByHeight: make(map[int][]byte),
	}
	proposerServer := &ProposerServer{
		chainStartFetcher: p,
		eth1InfoFetcher:   p,
		eth1BlockFetcher:  p,
		blockReceiver:     &mock.ChainService{State: beaconState},
		headFetcher:       &mock.ChainService{State: beaconState},
	}
	want := "could not fetch ETH1_FOLLOW_DISTANCE ancestor"
	if _, err := proposerServer.eth1Data(context.Background(), beaconState.Slot+1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error %v, received %v", want, err)
	}
}

func TestDefaultEth1Data_NoBlockExists(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	deps := []*depositcache.DepositContainer{
		{
			Index: 0,
			Block: big.NewInt(1000),
			Deposit: &ethpb.Deposit{
				Data: &ethpb.Deposit_Data{
					PublicKey:             []byte("a"),
					Signature:             make([]byte, 96),
					WithdrawalCredentials: make([]byte, 32),
				}},
		},
		{
			Index: 1,
			Block: big.NewInt(1200),
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
		depositCache.InsertDeposit(context.Background(), dp.Deposit, dp.Block, dp.Index, depositTrie.Root())
	}

	p := &mockPOW.POWChain{
		LatestBlockNumber: height,
		HashesByHeight: map[int][]byte{
			0:   []byte("hash0"),
			476: []byte("hash1024"),
		},
	}
	proposerServer := &ProposerServer{
		chainStartFetcher:      p,
		eth1InfoFetcher:        p,
		eth1BlockFetcher:       p,
		depositFetcher:         depositCache,
		pendingDepositsFetcher: depositCache,
	}

	defEth1Data := &ethpb.Eth1Data{
		DepositCount: 10,
		BlockHash:    []byte{'t', 'e', 's', 't'},
		DepositRoot:  []byte{'r', 'o', 'o', 't'},
	}

	p.Eth1Data = defEth1Data

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

	slot := uint64(10000)

	p := &mockPOW.POWChain{
		BlockNumberByHeight: map[uint64]*big.Int{
			60000: big.NewInt(4096),
		},
		HashesByHeight: map[int][]byte{
			3072: []byte("3072"),
		},
		Eth1Data: &ethpb.Eth1Data{
			DepositCount: 55,
		},
	}
	ps := &ProposerServer{
		chainStartFetcher: p,
		eth1InfoFetcher:   p,
		eth1BlockFetcher:  p,
		depositFetcher:    depositcache.NewDepositCache(),
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

func TestEth1Data_MockEnabled(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
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
	ps := &ProposerServer{
		headFetcher:   &mock.ChainService{State: &pbp2p.BeaconState{}},
		beaconDB:      db,
		mockEth1Votes: true,
	}
	headBlockRoot := [32]byte{1, 2, 3}
	headState := &pbp2p.BeaconState{
		Eth1DepositIndex: 64,
	}
	if err := db.SaveHeadBlockRoot(ctx, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, headState, headBlockRoot); err != nil {
		t.Fatal(err)
	}

	eth1Data, err := ps.eth1Data(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	wantedSlot := 100 % params.BeaconConfig().SlotsPerEth1VotingPeriod
	currentEpoch := helpers.SlotToEpoch(100)
	enc, err := ssz.Marshal(currentEpoch + wantedSlot)
	if err != nil {
		t.Fatal(err)
	}
	depRoot := hashutil.Hash(enc)
	blockHash := hashutil.Hash(depRoot[:])
	want := &ethpb.Eth1Data{
		DepositRoot: depRoot[:],
		BlockHash:   blockHash[:],
	}
	if !proto.Equal(eth1Data, want) {
		t.Errorf("Wanted %v, received %v", want, eth1Data)
	}
}

func Benchmark_Eth1Data(b *testing.B) {
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

	currentHeight := params.BeaconConfig().Eth1FollowDistance + 5
	p := &mockPOW.POWChain{
		LatestBlockNumber: big.NewInt(int64(currentHeight)),
		HashesByHeight:    hashesByHeight,
	}
	proposerServer := &ProposerServer{
		blockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		headFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		chainStartFetcher:      p,
		eth1InfoFetcher:        p,
		eth1BlockFetcher:       p,
		depositFetcher:         depositCache,
		pendingDepositsFetcher: depositCache,
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

	bs := &ProposerServer{
		blockReceiver:          &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		headFetcher:            &mock.ChainService{State: beaconState, Root: blkRoot[:]},
		chainStartFetcher:      p,
		eth1InfoFetcher:        p,
		eth1BlockFetcher:       p,
		depositFetcher:         depositCache,
		pendingDepositsFetcher: depositCache,
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
