package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	dbs "github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var closedContext = "context closed"
var mockSig [96]byte
var mockCreds [32]byte

type faultyPOWChainService struct {
	chainStartFeed *event.Feed
	hashesByHeight map[int][]byte
}

func (f *faultyPOWChainService) HasChainStarted() bool {
	return false
}
func (f *faultyPOWChainService) ETH2GenesisTime() uint64 {
	return 0
}

func (f *faultyPOWChainService) ChainStartFeed() *event.Feed {
	return f.chainStartFeed
}
func (f *faultyPOWChainService) LatestBlockHeight() *big.Int {
	return big.NewInt(0)
}

func (f *faultyPOWChainService) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	if f.hashesByHeight == nil {
		return false, big.NewInt(1), errors.New("failed")
	}

	return true, big.NewInt(1), nil
}

func (f *faultyPOWChainService) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	return [32]byte{}, errors.New("failed")
}

func (f *faultyPOWChainService) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	return 0, errors.New("failed")
}

func (f *faultyPOWChainService) DepositRoot() [32]byte {
	return [32]byte{}
}

func (f *faultyPOWChainService) DepositTrie() *trieutil.MerkleTrie {
	return &trieutil.MerkleTrie{}
}

func (f *faultyPOWChainService) ChainStartDeposits() []*pbp2p.Deposit {
	return []*pbp2p.Deposit{}
}

func (f *faultyPOWChainService) ChainStartDepositHashes() ([][]byte, error) {
	return [][]byte{}, errors.New("hashing failed")
}

type mockPOWChainService struct {
	chainStartFeed    *event.Feed
	latestBlockNumber *big.Int
	hashesByHeight    map[int][]byte
	blockTimeByHeight map[int]uint64
}

func (m *mockPOWChainService) HasChainStarted() bool {
	return true
}

func (m *mockPOWChainService) ETH2GenesisTime() uint64 {
	return uint64(time.Unix(0, 0).Unix())
}
func (m *mockPOWChainService) ChainStartFeed() *event.Feed {
	return m.chainStartFeed
}
func (m *mockPOWChainService) LatestBlockHeight() *big.Int {
	return m.latestBlockNumber
}

func (m *mockPOWChainService) DepositTrie() *trieutil.MerkleTrie {
	return &trieutil.MerkleTrie{}
}

func (m *mockPOWChainService) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	// Reverse the map of heights by hash.
	heightsByHash := make(map[[32]byte]int)
	for k, v := range m.hashesByHeight {
		h := bytesutil.ToBytes32(v)
		heightsByHash[h] = k
	}
	val, ok := heightsByHash[hash]
	if !ok {
		return false, nil, fmt.Errorf("could not fetch height for hash: %#x", hash)
	}
	return true, big.NewInt(int64(val)), nil
}

func (m *mockPOWChainService) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	k := int(height.Int64())
	val, ok := m.hashesByHeight[k]
	if !ok {
		return [32]byte{}, fmt.Errorf("could not fetch hash for height: %v", height)
	}
	return bytesutil.ToBytes32(val), nil
}

func (m *mockPOWChainService) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	h := int(height.Int64())
	return m.blockTimeByHeight[h], nil
}

func (m *mockPOWChainService) DepositRoot() [32]byte {
	root := []byte("depositroot")
	return bytesutil.ToBytes32(root)
}

func (m *mockPOWChainService) ChainStartDeposits() []*pbp2p.Deposit {
	return []*pbp2p.Deposit{}
}

func (m *mockPOWChainService) ChainStartDepositHashes() ([][]byte, error) {
	return [][]byte{}, nil
}

func TestWaitForChainStart_ContextClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx: ctx,
		powChainService: &faultyPOWChainService{
			chainStartFeed: new(event.Feed),
		},
		chainService: newMockChainService(),
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_WaitForChainStartServer(ctrl)
	go func(tt *testing.T) {
		if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); !strings.Contains(err.Error(), closedContext) {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForChainStart_AlreadyStarted(t *testing.T) {
	beaconServer := &BeaconServer{
		ctx: context.Background(),
		powChainService: &mockPOWChainService{
			chainStartFeed: new(event.Feed),
		},
		chainService: newMockChainService(),
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&pb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
}

func TestWaitForChainStart_NotStartedThenLogFired(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconServer := &BeaconServer{
		ctx:            context.Background(),
		chainStartChan: make(chan time.Time, 1),
		powChainService: &faultyPOWChainService{
			chainStartFeed: new(event.Feed),
		},
		chainService: newMockChainService(),
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&pb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	go func(tt *testing.T) {
		if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	beaconServer.chainStartChan <- time.Unix(0, 0)
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Sending ChainStart log and genesis time to connected validator clients")
}

func TestEth1Data_EmptyVotesFetchBlockHashFailure(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	beaconServer := &BeaconServer{
		beaconDB: db,
		powChainService: &faultyPOWChainService{
			hashesByHeight: make(map[int][]byte),
		},
	}
	beaconState := &pbp2p.BeaconState{
		Eth1Data: &pbp2p.Eth1Data{
			BlockHash: []byte{'a'},
		},
		Eth1DataVotes: []*pbp2p.Eth1Data{},
	}
	if err := beaconServer.beaconDB.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	want := "could not fetch ETH1_FOLLOW_DISTANCE ancestor"
	if _, err := beaconServer.eth1Data(context.Background()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error %v, received %v", want, err)
	}
}

func TestEth1Data_EmptyVotesOk(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	deps := []*dbs.DepositContainer{
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
	depsData := [][]byte{}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("could not setup deposit trie: %v", err)
	}
	for _, dp := range deps {
		db.InsertDeposit(context.Background(), dp.Deposit, big.NewInt(0), dp.Index, depositTrie.Root())
		depHash, err := hashutil.DepositHash(dp.Deposit.Data)
		if err != nil {
			t.Errorf("Could not hash deposit")
		}
		depsData = append(depsData, depHash[:])
	}
	depositRoot := depositTrie.Root()
	beaconState := &pbp2p.BeaconState{
		Eth1Data: &pbp2p.Eth1Data{
			BlockHash:   []byte("hash0"),
			DepositRoot: depositRoot[:],
		},
		Eth1DataVotes: []*pbp2p.Eth1Data{},
	}

	powChainService := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			0: []byte("hash0"),
			1: beaconState.Eth1Data.DepositRoot,
		},
	}
	beaconServer := &BeaconServer{
		beaconDB:        db,
		powChainService: powChainService,
	}

	if err := beaconServer.beaconDB.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	result, err := beaconServer.eth1Data(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// If the data vote objects are empty, the deposit root should be the one corresponding
	// to the deposit contract in the powchain service, fetched using powChainService.DepositRoot()
	if !bytes.Equal(result.DepositRoot, depositRoot[:]) {
		t.Errorf(
			"Expected deposit roots to match, received %#x == %#x",
			result.DepositRoot,
			depositRoot,
		)
	}
}

func TestEth1Data_NonEmptyVotesSelectsBestVote(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	ctx := context.Background()
	eth1DataVotes := []*pbp2p.Eth1Data{
		{
			BlockHash:    []byte("block0"),
			DepositRoot:  []byte("deposit0001234567890123456789012"),
			DepositCount: 2,
		},
		{
			BlockHash:    []byte("block1"),
			DepositRoot:  []byte("deposit1001234567890123456789012"),
			DepositCount: 2,
		},
		{
			BlockHash:    []byte("block1"),
			DepositRoot:  []byte("deposit1001234567890123456789012"),
			DepositCount: 2,
		},
		// We include the case in which the vote counts might match, and in that
		// case we break ties by checking which block hash has the highest
		// block height in the eth1.0 chain.
		{
			BlockHash:    []byte("block2"),
			DepositRoot:  []byte("deposit2001234567890123456789012"),
			DepositCount: 2,
		},
		{
			BlockHash:    []byte("block2"),
			DepositRoot:  []byte("deposit2001234567890123456789012"),
			DepositCount: 2,
		},
		{
			BlockHash:    []byte("block2"),
			DepositRoot:  []byte("deposit2001234567890123456789012"),
			DepositCount: 2,
		},
		{
			BlockHash:    []byte("block4"),
			DepositRoot:  []byte("deposit3001234567890123456789012"),
			DepositCount: 2,
		},
		{
			BlockHash:    []byte("block4"),
			DepositRoot:  []byte("deposit3001234567890123456789012"),
			DepositCount: 2,
		},
		{
			BlockHash:    []byte("block4"),
			DepositRoot:  []byte("deposit3001234567890123456789012"),
			DepositCount: 2,
		},
		// We include a case with higher vote count but wrong deposit count
		// that shouldnt be counted at all.
		{
			BlockHash:    []byte("block4"),
			DepositRoot:  []byte("deposit4001234567890123456789012"),
			DepositCount: 1,
		},
		{
			BlockHash:    []byte("block4"),
			DepositRoot:  []byte("deposit4001234567890123456789012"),
			DepositCount: 1,
		},
		{
			BlockHash:    []byte("block4"),
			DepositRoot:  []byte("deposit4001234567890123456789012"),
			DepositCount: 1,
		},
		{
			BlockHash:    []byte("block4"),
			DepositRoot:  []byte("deposit4001234567890123456789012"),
			DepositCount: 1,
		},
	}

	var mockSig [96]byte
	var mockCreds [32]byte
	deposits := []*dbs.DepositContainer{
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

	for _, dp := range deposits {
		var root [32]byte
		copy(root[:], eth1DataVotes[dp.Index].DepositRoot)
		db.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, root)
	}
	beaconState := &pbp2p.BeaconState{
		Eth1DataVotes: eth1DataVotes,
		Eth1Data: &pbp2p.Eth1Data{
			BlockHash:   []byte("stub"),
			DepositRoot: []byte("first"),
		},
	}
	if err := db.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	currentHeight := params.BeaconConfig().Eth1FollowDistance + 9
	beaconServer := &BeaconServer{
		beaconDB: db,
		powChainService: &mockPOWChainService{
			latestBlockNumber: big.NewInt(int64(currentHeight)),
			hashesByHeight: map[int][]byte{
				0: beaconState.Eth1Data.DepositRoot,
				// adding some not relevant blocks heights to test that search works
				1: {1},
				2: beaconState.Eth1DataVotes[0].BlockHash,
				3: {3},
				4: beaconState.Eth1DataVotes[1].BlockHash,
				5: {5},
				6: beaconState.Eth1DataVotes[3].BlockHash,
				7: {7},
				// We will give the hash at index 2 in the beacon state's latest eth1 votes
				// priority in being selected as the best vote by giving it the highest block number.
				8: beaconState.Eth1DataVotes[2].BlockHash,
				9: {9},
			},
		},
	}

	eth1data, err := beaconServer.eth1Data(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Vote at index 2 should have won the best vote selection mechanism as it had the highest block number
	// despite being tied at vote count with the vote at index 3.
	if !bytes.Equal(eth1data.BlockHash, beaconState.Eth1DataVotes[2].BlockHash) {
		t.Errorf(
			"Expected block hashes to match, received %#x == %#x",
			eth1data.BlockHash,
			beaconState.Eth1DataVotes[2].BlockHash,
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

func Benchmark_Eth1Data(b *testing.B) {
	db := internal.SetupDB(b)
	defer internal.TeardownDB(b, db)
	ctx := context.Background()

	hashesByHeight := make(map[int][]byte)

	beaconState := &pbp2p.BeaconState{
		Eth1DataVotes: []*pbp2p.Eth1Data{},
		Eth1Data: &pbp2p.Eth1Data{
			BlockHash: []byte("stub"),
		},
	}
	var mockSig [96]byte
	var mockCreds [32]byte
	deposits := []*dbs.DepositContainer{
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

	for i, dp := range deposits {
		var root [32]byte
		copy(root[:], []byte{'d', 'e', 'p', 'o', 's', 'i', 't', byte(i)})
		db.InsertDeposit(ctx, dp.Deposit, big.NewInt(int64(dp.Index)), dp.Index, root)
	}
	numOfVotes := 1000
	for i := 0; i < numOfVotes; i++ {
		blockhash := []byte{'b', 'l', 'o', 'c', 'k', byte(i)}
		deposit := []byte{'d', 'e', 'p', 'o', 's', 'i', 't', byte(i)}
		beaconState.Eth1DataVotes = append(beaconState.Eth1DataVotes, &pbp2p.Eth1Data{
			BlockHash:   blockhash,
			DepositRoot: deposit,
		})
		hashesByHeight[i] = blockhash
	}
	hashesByHeight[numOfVotes+1] = []byte("stub")

	if err := db.SaveState(ctx, beaconState); err != nil {
		b.Fatal(err)
	}
	currentHeight := params.BeaconConfig().Eth1FollowDistance + 5
	beaconServer := &BeaconServer{
		beaconDB: db,
		powChainService: &mockPOWChainService{
			latestBlockNumber: big.NewInt(int64(currentHeight)),
			hashesByHeight:    hashesByHeight,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := beaconServer.eth1Data(context.Background())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestBlockTree_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()
	// We want to ensure that if our block tree looks as follows, the RPC response
	// returns the correct information.
	//                   /->[A, Slot 3, 3 Votes]->[B, Slot 4, 3 Votes]
	// [Justified Block]->[C, Slot 3, 2 Votes]
	//                   \->[D, Slot 3, 2 Votes]->[SKIP SLOT]->[E, Slot 5, 1 Vote]
	var validators []*pbp2p.Validator
	for i := 0; i < 13; i++ {
		validators = append(validators, &pbp2p.Validator{ExitEpoch: params.BeaconConfig().FarFutureEpoch})
	}
	justifiedState := &pbp2p.BeaconState{
		Slot:       0,
		Balances:   make([]uint64, 11),
		Validators: validators,
	}
	for i := 0; i < len(justifiedState.Balances); i++ {
		justifiedState.Balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	if err := db.SaveJustifiedState(justifiedState); err != nil {
		t.Fatal(err)
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := blockutil.BlockSigningRoot(justifiedBlock)

	balances := []uint64{params.BeaconConfig().MaxEffectiveBalance}
	b1 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
		StateRoot:  []byte{0x1},
	}
	b1Root, _ := blockutil.BlockSigningRoot(b1)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b1Root); err != nil {
		t.Fatal(err)
	}
	b2 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
		StateRoot:  []byte{0x2},
	}
	b2Root, _ := blockutil.BlockSigningRoot(b2)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
		StateRoot:  []byte{0x3},
	}
	b3Root, _ := blockutil.BlockSigningRoot(b3)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b3Root); err != nil {
		t.Fatal(err)
	}
	b4 := &pbp2p.BeaconBlock{
		Slot:       4,
		ParentRoot: b1Root[:],
		StateRoot:  []byte{0x4},
	}
	b4Root, _ := blockutil.BlockSigningRoot(b4)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       4,
		Validators: validators,
		Balances:   balances,
	}, b4Root); err != nil {
		t.Fatal(err)
	}
	b5 := &pbp2p.BeaconBlock{
		Slot:       5,
		ParentRoot: b3Root[:],
		StateRoot:  []byte{0x5},
	}
	b5Root, _ := blockutil.BlockSigningRoot(b5)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       5,
		Validators: validators,
		Balances:   balances,
	}, b5Root); err != nil {
		t.Fatal(err)
	}
	attestationTargets := make(map[uint64]*pbp2p.AttestationTarget)
	// We give block A 3 votes.
	attestationTargets[0] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRoot,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[1] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRoot,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[2] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRoot,
		BlockRoot:  b1Root[:],
	}

	// We give block C 2 votes.
	attestationTargets[3] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRoot,
		BlockRoot:  b2Root[:],
	}
	attestationTargets[4] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRoot,
		BlockRoot:  b2Root[:],
	}

	// We give block D 2 votes.
	attestationTargets[5] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRoot,
		BlockRoot:  b3Root[:],
	}
	attestationTargets[6] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRoot,
		BlockRoot:  b3Root[:],
	}

	// We give block B 3 votes.
	attestationTargets[7] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRoot,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[8] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRoot,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[9] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRoot,
		BlockRoot:  b4Root[:],
	}

	// We give block E 1 vote.
	attestationTargets[10] = &pbp2p.AttestationTarget{
		Slot:       b5.Slot,
		ParentRoot: b5.ParentRoot,
		BlockRoot:  b5Root[:],
	}

	tree := []*pb.BlockTreeResponse_TreeNode{
		{
			Block:             b1,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b2,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b3,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b4,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b5,
			ParticipatedVotes: 1 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
	}
	for _, node := range tree {
		if err := db.SaveBlock(node.Block); err != nil {
			t.Fatal(err)
		}
	}

	headState := &pbp2p.BeaconState{
		Slot: b4.Slot,
	}
	if err := db.UpdateChainHead(ctx, b4, headState); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconServer{
		beaconDB:       db,
		targetsFetcher: &mockChainService{targets: attestationTargets},
	}
	sort.Slice(tree, func(i, j int) bool {
		return string(tree[i].Block.StateRoot) < string(tree[j].Block.StateRoot)
	})

	resp, err := bs.BlockTree(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Tree) != 4 {
		t.Errorf("Wanted len %d, received %d", 4, len(resp.Tree))
	}
}

func TestBlockTreeBySlots_ArgsValildation(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()
	// We want to ensure that if our block tree looks as follows, the RPC response
	// returns the correct information.
	//                   /->[A, Slot 3, 3 Votes]->[B, Slot 4, 3 Votes]
	// [Justified Block]->[C, Slot 3, 2 Votes]
	//                   \->[D, Slot 3, 2 Votes]->[SKIP SLOT]->[E, Slot 5, 1 Vote]
	justifiedState := &pbp2p.BeaconState{
		Slot:     0,
		Balances: make([]uint64, 11),
	}
	for i := 0; i < len(justifiedState.Balances); i++ {
		justifiedState.Balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	if err := db.SaveJustifiedState(justifiedState); err != nil {
		t.Fatal(err)
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := blockutil.BlockSigningRoot(justifiedBlock)
	validators := []*pbp2p.Validator{{ExitEpoch: params.BeaconConfig().FarFutureEpoch}}
	balances := []uint64{params.BeaconConfig().MaxEffectiveBalance}
	b1 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b1Root, _ := blockutil.BlockSigningRoot(b1)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b1Root); err != nil {
		t.Fatal(err)
	}
	b2 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b2Root, _ := blockutil.BlockSigningRoot(b2)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b3Root, _ := blockutil.BlockSigningRoot(b3)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b3Root); err != nil {
		t.Fatal(err)
	}
	b4 := &pbp2p.BeaconBlock{
		Slot:       4,
		ParentRoot: b1Root[:],
	}
	b4Root, _ := blockutil.BlockSigningRoot(b4)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       4,
		Validators: validators,
		Balances:   balances,
	}, b4Root); err != nil {
		t.Fatal(err)
	}
	b5 := &pbp2p.BeaconBlock{
		Slot:       5,
		ParentRoot: b3Root[:],
	}
	b5Root, _ := blockutil.BlockSigningRoot(b5)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       5,
		Validators: validators,
		Balances:   balances,
	}, b5Root); err != nil {
		t.Fatal(err)
	}
	attestationTargets := make(map[uint64]*pbp2p.AttestationTarget)
	// We give block A 3 votes.
	attestationTargets[0] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRoot,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[1] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRoot,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[2] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRoot,
		BlockRoot:  b1Root[:],
	}

	// We give block C 2 votes.
	attestationTargets[3] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRoot,
		BlockRoot:  b2Root[:],
	}
	attestationTargets[4] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRoot,
		BlockRoot:  b2Root[:],
	}

	// We give block D 2 votes.
	attestationTargets[5] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRoot,
		BlockRoot:  b3Root[:],
	}
	attestationTargets[6] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRoot,
		BlockRoot:  b3Root[:],
	}

	// We give block B 3 votes.
	attestationTargets[7] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRoot,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[8] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRoot,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[9] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRoot,
		BlockRoot:  b4Root[:],
	}

	// We give block E 1 vote.
	attestationTargets[10] = &pbp2p.AttestationTarget{
		Slot:       b5.Slot,
		ParentRoot: b5.ParentRoot,
		BlockRoot:  b5Root[:],
	}

	tree := []*pb.BlockTreeResponse_TreeNode{
		{
			Block:             b1,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b2,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b3,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b4,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b5,
			ParticipatedVotes: 1 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
	}
	for _, node := range tree {
		if err := db.SaveBlock(node.Block); err != nil {
			t.Fatal(err)
		}
	}
	headState := &pbp2p.BeaconState{
		Slot: b4.Slot,
	}
	if err := db.UpdateChainHead(ctx, b4, headState); err != nil {
		t.Fatal(err)
	}
	bs := &BeaconServer{
		beaconDB:       db,
		targetsFetcher: &mockChainService{targets: attestationTargets},
	}
	if _, err := bs.BlockTreeBySlots(ctx, nil); err == nil {
		// There should be a "argument 'TreeBlockSlotRequest' cannot be nil" error
		t.Fatal(err)
	}
	slotRange := &pb.TreeBlockSlotRequest{
		SlotFrom: 4,
		SlotTo:   3,
	}
	if _, err := bs.BlockTreeBySlots(ctx, slotRange); err == nil {
		// There should be a 'Upper limit of slot range cannot be lower than the lower limit' error.
		t.Fatal(err)
	}
}
func TestBlockTreeBySlots_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()
	// We want to ensure that if our block tree looks as follows, the RPC response
	// returns the correct information.
	//                   /->[A, Slot 3, 3 Votes]->[B, Slot 4, 3 Votes]
	// [Justified Block]->[C, Slot 3, 2 Votes]
	//                   \->[D, Slot 3, 2 Votes]->[SKIP SLOT]->[E, Slot 5, 1 Vote]
	justifiedState := &pbp2p.BeaconState{
		Slot:     0,
		Balances: make([]uint64, 11),
	}
	for i := 0; i < len(justifiedState.Balances); i++ {
		justifiedState.Balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	var validators []*pbp2p.Validator
	for i := 0; i < 11; i++ {
		validators = append(validators, &pbp2p.Validator{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance})
	}
	justifiedState.Validators = validators
	if err := db.SaveJustifiedState(justifiedState); err != nil {
		t.Fatal(err)
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := blockutil.BlockSigningRoot(justifiedBlock)
	balances := []uint64{params.BeaconConfig().MaxEffectiveBalance}
	b1 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b1Root, _ := blockutil.BlockSigningRoot(b1)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b1Root); err != nil {
		t.Fatal(err)
	}
	b2 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b2Root, _ := blockutil.BlockSigningRoot(b2)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b3Root, _ := blockutil.BlockSigningRoot(b3)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b3Root); err != nil {
		t.Fatal(err)
	}
	b4 := &pbp2p.BeaconBlock{
		Slot:       4,
		ParentRoot: b1Root[:],
	}
	b4Root, _ := blockutil.BlockSigningRoot(b4)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       4,
		Validators: validators,
		Balances:   balances,
	}, b4Root); err != nil {
		t.Fatal(err)
	}
	b5 := &pbp2p.BeaconBlock{
		Slot:       5,
		ParentRoot: b3Root[:],
	}
	b5Root, _ := blockutil.BlockSigningRoot(b5)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       5,
		Validators: validators,
		Balances:   balances,
	}, b5Root); err != nil {
		t.Fatal(err)
	}
	attestationTargets := make(map[uint64]*pbp2p.AttestationTarget)
	// We give block A 3 votes.
	attestationTargets[0] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRoot,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[1] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRoot,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[2] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRoot,
		BlockRoot:  b1Root[:],
	}

	// We give block C 2 votes.
	attestationTargets[3] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRoot,
		BlockRoot:  b2Root[:],
	}
	attestationTargets[4] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRoot,
		BlockRoot:  b2Root[:],
	}

	// We give block D 2 votes.
	attestationTargets[5] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRoot,
		BlockRoot:  b3Root[:],
	}
	attestationTargets[6] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRoot,
		BlockRoot:  b3Root[:],
	}

	// We give block B 3 votes.
	attestationTargets[7] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRoot,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[8] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRoot,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[9] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRoot,
		BlockRoot:  b4Root[:],
	}

	// We give block E 1 vote.
	attestationTargets[10] = &pbp2p.AttestationTarget{
		Slot:       b5.Slot,
		ParentRoot: b5.ParentRoot,
		BlockRoot:  b5Root[:],
	}

	tree := []*pb.BlockTreeResponse_TreeNode{
		{
			Block:             b1,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b2,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b3,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b4,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
		{
			Block:             b5,
			ParticipatedVotes: 1 * params.BeaconConfig().MaxEffectiveBalance,
			TotalVotes:        params.BeaconConfig().MaxEffectiveBalance,
		},
	}
	for _, node := range tree {
		if err := db.SaveBlock(node.Block); err != nil {
			t.Fatal(err)
		}
	}

	headState := &pbp2p.BeaconState{
		Slot: b4.Slot,
	}
	if err := db.UpdateChainHead(ctx, b4, headState); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconServer{
		beaconDB:       db,
		targetsFetcher: &mockChainService{targets: attestationTargets},
	}
	slotRange := &pb.TreeBlockSlotRequest{
		SlotFrom: 3,
		SlotTo:   4,
	}
	resp, err := bs.BlockTreeBySlots(ctx, slotRange)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Tree) != 2 {
		t.Logf("Incorrect number of nodes in tree, expected: %d, actual: %d", 2, len(resp.Tree))
	}
}
