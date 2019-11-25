package proposer

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

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
	proposerServer := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		BlockReceiver:     &mock.ChainService{State: beaconState},
		HeadFetcher:       &mock.ChainService{State: beaconState},
	}
	want := "could not fetch ETH1_FOLLOW_DISTANCE ancestor"
	if _, err := proposerServer.getEth1Data(context.Background(), beaconState.Slot+1); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error %v, received %v", want, err)
	}
}

func TestEth1Data(t *testing.T) {
	slot := uint64(10000)
	beaconState := &pbp2p.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			BlockHash: []byte{'a'},
		},
		Eth1DataVotes: []*ethpb.Eth1Data{},
	}

	p := &mockPOW.POWChain{
		BlockNumberByHeight: map[uint64]*big.Int{
			slot * params.BeaconConfig().SecondsPerSlot: big.NewInt(4096),
		},
		HashesByHeight: map[int][]byte{
			3072: []byte("3072"),
		},
		Eth1Data: &ethpb.Eth1Data{
			DepositCount: 55,
		},
	}
	ps := &Server{
		ChainStartFetcher: p,
		Eth1InfoFetcher:   p,
		Eth1BlockFetcher:  p,
		DepositFetcher:    depositcache.NewDepositCache(),
		HeadFetcher:       &mock.ChainService{State: beaconState},
	}

	ctx := context.Background()
	eth1Data, err := ps.getEth1Data(ctx, slot)
	if err != nil {
		t.Fatal(err)
	}

	if eth1Data == nil || eth1Data.DepositCount != 55 {
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
	ps := &Server{
		HeadFetcher:   &mock.ChainService{State: &pbp2p.BeaconState{}},
		BeaconDB:      db,
		MockEth1Votes: true,
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

	eth1Data, err := ps.getEth1Data(ctx, 100)
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
		_, err := proposerServer.getEth1Data(context.Background(), beaconState.Slot+1)
		if err != nil {
			b.Fatal(err)
		}
	}
}
