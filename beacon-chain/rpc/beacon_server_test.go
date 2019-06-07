package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var closedContext = "context closed"

type faultyPOWChainService struct {
	chainStartFeed *event.Feed
	hashesByHeight map[int][]byte
}

func (f *faultyPOWChainService) HasChainStartLogOccurred() (bool, uint64, error) {
	return false, uint64(time.Now().Unix()), nil
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

func (f *faultyPOWChainService) DepositRoot() [32]byte {
	return [32]byte{}
}

func (f *faultyPOWChainService) DepositTrie() *trieutil.MerkleTrie {
	return &trieutil.MerkleTrie{}
}

func (f *faultyPOWChainService) ChainStartDeposits() [][]byte {
	return [][]byte{}
}

type mockPOWChainService struct {
	chainStartFeed    *event.Feed
	latestBlockNumber *big.Int
	hashesByHeight    map[int][]byte
}

func (m *mockPOWChainService) HasChainStartLogOccurred() (bool, uint64, error) {
	return true, uint64(time.Unix(0, 0).Unix()), nil
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

func (m *mockPOWChainService) DepositRoot() [32]byte {
	root := []byte("depositroot")
	return bytesutil.ToBytes32(root)
}

func (m *mockPOWChainService) ChainStartDeposits() [][]byte {
	return [][]byte{}
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

func TestBlockTree_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	// We want to ensure that if our block tree looks as follows, the RPC response
	// returns the correct information.
	//                   /->[A, Slot 3, 3 Votes]->[B, Slot 4, 3 Votes]
	// [Justified Block]->[C, Slot 3, 2 Votes]
	//                   \->[D, Slot 3, 2 Votes]->[SKIP SLOT]->[E, Slot 5, 1 Vote]
	var validators []*pbp2p.Validator
	for i := 0; i < 11; i++ {
		validators = append(validators, &pbp2p.Validator{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance})
	}
	justifiedState := &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: validators,
	}

	if err := db.SaveJustifiedState(justifiedState); err != nil {
		t.Fatal(err)
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: params.BeaconConfig().GenesisSlot,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := hashutil.HashBeaconBlock(justifiedBlock)
	b1 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
		Body: &pbp2p.BeaconBlockBody{
			RandaoReveal: []byte("D"),
		},
	}
	b1Root, _ := hashutil.HashBeaconBlock(b1)
	b2 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
		Body: &pbp2p.BeaconBlockBody{
			RandaoReveal: []byte("C"),
		}}
	b2Root, _ := hashutil.HashBeaconBlock(b2)
	b3 := &pbp2p.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
		Body: &pbp2p.BeaconBlockBody{
			RandaoReveal: []byte("A"),
		}}
	b3Root, _ := hashutil.HashBeaconBlock(b1)
	b4 := &pbp2p.BeaconBlock{
		Slot:       4,
		ParentRoot: b1Root[:],
		Body: &pbp2p.BeaconBlockBody{
			RandaoReveal: []byte("B"),
		}}
	b4Root, _ := hashutil.HashBeaconBlock(b4)
	b5 := &pbp2p.BeaconBlock{
		Slot:       5,
		ParentRoot: b3Root[:],
		Body: &pbp2p.BeaconBlockBody{
			RandaoReveal: []byte("E"),
		}}
	b5Root, _ := hashutil.HashBeaconBlock(b5)

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
			Block: b1,
			Votes: 3 * params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block: b2,
			Votes: 2 * params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block: b3,
			Votes: 2 * params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block: b4,
			Votes: 3 * params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block: b5,
			Votes: 1 * params.BeaconConfig().MaxDepositAmount,
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
	ctx := context.Background()
	if err := db.UpdateChainHead(ctx, b4, headState); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconServer{
		beaconDB:       db,
		targetsFetcher: &mockChainService{targets: attestationTargets},
	}
	resp, err := bs.BlockTree(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(resp.Tree, func(i, j int) bool {
		return string(resp.Tree[i].Block.Signature) < string(resp.Tree[j].Block.Signature)
	})
	sort.Slice(tree, func(i, j int) bool {
		return string(tree[i].Block.Signature) < string(tree[j].Block.Signature)
	})
	for i := range resp.Tree {
		if !proto.Equal(resp.Tree[i].Block, tree[i].Block) {
			t.Errorf("Expected %v, received %v", tree[i].Block, resp.Tree[i].Block)
		}
	}
}
