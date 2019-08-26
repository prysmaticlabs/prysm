package slasher

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
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var closedContext = "context closed"
var mockSig [96]byte
var mockCreds [32]byte

func TestWaitForChainStart_ContextClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	slasherServer := &Server{
		ctx: ctx,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_WaitForChainStartServer(ctrl)
	go func(tt *testing.T) {
		if err := slasherServer.IsSlashableAttestation(&ptypes.Empty{}, mockStream); !strings.Contains(err.Error(), closedContext) {
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
		chainService: &mockStateFeedListener{},
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
		chainService: &mockStateFeedListener{},
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
	t.Skip() // TODO(3219): Add after new fork choice service.
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()
	// We want to ensure that if our block tree looks as follows, the RPC response
	// returns the correct information.
	//                   /->[A, Slot 3, 3 Votes]->[B, Slot 4, 3 Votes]
	// [Justified Block]->[C, Slot 3, 2 Votes]
	//                   \->[D, Slot 3, 2 Votes]->[SKIP SLOT]->[E, Slot 5, 1 Vote]
	var validators []*ethpb.Validator
	for i := 0; i < 13; i++ {
		validators = append(validators, &ethpb.Validator{ExitEpoch: params.BeaconConfig().FarFutureEpoch})
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
	justifiedBlock := &ethpb.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := ssz.SigningRoot(justifiedBlock)

	balances := []uint64{params.BeaconConfig().MaxEffectiveBalance}
	b1 := &ethpb.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
		StateRoot:  []byte{0x1},
	}
	b1Root, _ := ssz.SigningRoot(b1)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b1Root); err != nil {
		t.Fatal(err)
	}
	b2 := &ethpb.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
		StateRoot:  []byte{0x2},
	}
	b2Root, _ := ssz.SigningRoot(b2)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
		StateRoot:  []byte{0x3},
	}
	b3Root, _ := ssz.SigningRoot(b3)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b3Root); err != nil {
		t.Fatal(err)
	}
	b4 := &ethpb.BeaconBlock{
		Slot:       4,
		ParentRoot: b1Root[:],
		StateRoot:  []byte{0x4},
	}
	b4Root, _ := ssz.SigningRoot(b4)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       4,
		Validators: validators,
		Balances:   balances,
	}, b4Root); err != nil {
		t.Fatal(err)
	}
	b5 := &ethpb.BeaconBlock{
		Slot:       5,
		ParentRoot: b3Root[:],
		StateRoot:  []byte{0x5},
	}
	b5Root, _ := ssz.SigningRoot(b5)
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
		Slot:            b1.Slot,
		ParentRoot:      b1.ParentRoot,
		BeaconBlockRoot: b1Root[:],
	}
	attestationTargets[1] = &pbp2p.AttestationTarget{
		Slot:            b1.Slot,
		ParentRoot:      b1.ParentRoot,
		BeaconBlockRoot: b1Root[:],
	}
	attestationTargets[2] = &pbp2p.AttestationTarget{
		Slot:            b1.Slot,
		ParentRoot:      b1.ParentRoot,
		BeaconBlockRoot: b1Root[:],
	}

	// We give block C 2 votes.
	attestationTargets[3] = &pbp2p.AttestationTarget{
		Slot:            b2.Slot,
		ParentRoot:      b2.ParentRoot,
		BeaconBlockRoot: b2Root[:],
	}
	attestationTargets[4] = &pbp2p.AttestationTarget{
		Slot:            b2.Slot,
		ParentRoot:      b2.ParentRoot,
		BeaconBlockRoot: b2Root[:],
	}

	// We give block D 2 votes.
	attestationTargets[5] = &pbp2p.AttestationTarget{
		Slot:            b3.Slot,
		ParentRoot:      b3.ParentRoot,
		BeaconBlockRoot: b3Root[:],
	}
	attestationTargets[6] = &pbp2p.AttestationTarget{
		Slot:            b3.Slot,
		ParentRoot:      b3.ParentRoot,
		BeaconBlockRoot: b3Root[:],
	}

	// We give block B 3 votes.
	attestationTargets[7] = &pbp2p.AttestationTarget{
		Slot:            b4.Slot,
		ParentRoot:      b4.ParentRoot,
		BeaconBlockRoot: b4Root[:],
	}
	attestationTargets[8] = &pbp2p.AttestationTarget{
		Slot:            b4.Slot,
		ParentRoot:      b4.ParentRoot,
		BeaconBlockRoot: b4Root[:],
	}
	attestationTargets[9] = &pbp2p.AttestationTarget{
		Slot:            b4.Slot,
		ParentRoot:      b4.ParentRoot,
		BeaconBlockRoot: b4Root[:],
	}

	// We give block E 1 vote.
	attestationTargets[10] = &pbp2p.AttestationTarget{
		Slot:            b5.Slot,
		ParentRoot:      b5.ParentRoot,
		BeaconBlockRoot: b5Root[:],
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
		if err := db.SaveBlockDeprecated(node.Block); err != nil {
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
		beaconDB: db,
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
	t.Skip() // TODO(3219): Add after new fork choice service.
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
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
	justifiedBlock := &ethpb.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := ssz.SigningRoot(justifiedBlock)
	validators := []*ethpb.Validator{{ExitEpoch: params.BeaconConfig().FarFutureEpoch}}
	balances := []uint64{params.BeaconConfig().MaxEffectiveBalance}
	b1 := &ethpb.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b1Root, _ := ssz.SigningRoot(b1)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b1Root); err != nil {
		t.Fatal(err)
	}
	b2 := &ethpb.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b2Root, _ := ssz.SigningRoot(b2)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b3Root, _ := ssz.SigningRoot(b3)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b3Root); err != nil {
		t.Fatal(err)
	}
	b4 := &ethpb.BeaconBlock{
		Slot:       4,
		ParentRoot: b1Root[:],
	}
	b4Root, _ := ssz.SigningRoot(b4)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       4,
		Validators: validators,
		Balances:   balances,
	}, b4Root); err != nil {
		t.Fatal(err)
	}
	b5 := &ethpb.BeaconBlock{
		Slot:       5,
		ParentRoot: b3Root[:],
	}
	b5Root, _ := ssz.SigningRoot(b5)
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
		Slot:            b1.Slot,
		ParentRoot:      b1.ParentRoot,
		BeaconBlockRoot: b1Root[:],
	}
	attestationTargets[1] = &pbp2p.AttestationTarget{
		Slot:            b1.Slot,
		ParentRoot:      b1.ParentRoot,
		BeaconBlockRoot: b1Root[:],
	}
	attestationTargets[2] = &pbp2p.AttestationTarget{
		Slot:            b1.Slot,
		ParentRoot:      b1.ParentRoot,
		BeaconBlockRoot: b1Root[:],
	}

	// We give block C 2 votes.
	attestationTargets[3] = &pbp2p.AttestationTarget{
		Slot:            b2.Slot,
		ParentRoot:      b2.ParentRoot,
		BeaconBlockRoot: b2Root[:],
	}
	attestationTargets[4] = &pbp2p.AttestationTarget{
		Slot:            b2.Slot,
		ParentRoot:      b2.ParentRoot,
		BeaconBlockRoot: b2Root[:],
	}

	// We give block D 2 votes.
	attestationTargets[5] = &pbp2p.AttestationTarget{
		Slot:            b3.Slot,
		ParentRoot:      b3.ParentRoot,
		BeaconBlockRoot: b3Root[:],
	}
	attestationTargets[6] = &pbp2p.AttestationTarget{
		Slot:            b3.Slot,
		ParentRoot:      b3.ParentRoot,
		BeaconBlockRoot: b3Root[:],
	}

	// We give block B 3 votes.
	attestationTargets[7] = &pbp2p.AttestationTarget{
		Slot:            b4.Slot,
		ParentRoot:      b4.ParentRoot,
		BeaconBlockRoot: b4Root[:],
	}
	attestationTargets[8] = &pbp2p.AttestationTarget{
		Slot:            b4.Slot,
		ParentRoot:      b4.ParentRoot,
		BeaconBlockRoot: b4Root[:],
	}
	attestationTargets[9] = &pbp2p.AttestationTarget{
		Slot:            b4.Slot,
		ParentRoot:      b4.ParentRoot,
		BeaconBlockRoot: b4Root[:],
	}

	// We give block E 1 vote.
	attestationTargets[10] = &pbp2p.AttestationTarget{
		Slot:            b5.Slot,
		ParentRoot:      b5.ParentRoot,
		BeaconBlockRoot: b5Root[:],
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
		if err := db.SaveBlockDeprecated(node.Block); err != nil {
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
		beaconDB: db,
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
	t.Skip() // TODO(3219): Add after new fork choice service.
	helpers.ClearAllCaches()

	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
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
	var validators []*ethpb.Validator
	for i := 0; i < 11; i++ {
		validators = append(validators, &ethpb.Validator{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance})
	}
	justifiedState.Validators = validators
	if err := db.SaveJustifiedState(justifiedState); err != nil {
		t.Fatal(err)
	}
	justifiedBlock := &ethpb.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := ssz.SigningRoot(justifiedBlock)
	balances := []uint64{params.BeaconConfig().MaxEffectiveBalance}
	b1 := &ethpb.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b1Root, _ := ssz.SigningRoot(b1)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b1Root); err != nil {
		t.Fatal(err)
	}
	b2 := &ethpb.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b2Root, _ := ssz.SigningRoot(b2)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.BeaconBlock{
		Slot:       3,
		ParentRoot: justifiedRoot[:],
	}
	b3Root, _ := ssz.SigningRoot(b3)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       3,
		Validators: validators,
		Balances:   balances,
	}, b3Root); err != nil {
		t.Fatal(err)
	}
	b4 := &ethpb.BeaconBlock{
		Slot:       4,
		ParentRoot: b1Root[:],
	}
	b4Root, _ := ssz.SigningRoot(b4)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:       4,
		Validators: validators,
		Balances:   balances,
	}, b4Root); err != nil {
		t.Fatal(err)
	}
	b5 := &ethpb.BeaconBlock{
		Slot:       5,
		ParentRoot: b3Root[:],
	}
	b5Root, _ := ssz.SigningRoot(b5)
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
		Slot:            b1.Slot,
		ParentRoot:      b1.ParentRoot,
		BeaconBlockRoot: b1Root[:],
	}
	attestationTargets[1] = &pbp2p.AttestationTarget{
		Slot:            b1.Slot,
		ParentRoot:      b1.ParentRoot,
		BeaconBlockRoot: b1Root[:],
	}
	attestationTargets[2] = &pbp2p.AttestationTarget{
		Slot:            b1.Slot,
		ParentRoot:      b1.ParentRoot,
		BeaconBlockRoot: b1Root[:],
	}

	// We give block C 2 votes.
	attestationTargets[3] = &pbp2p.AttestationTarget{
		Slot:            b2.Slot,
		ParentRoot:      b2.ParentRoot,
		BeaconBlockRoot: b2Root[:],
	}
	attestationTargets[4] = &pbp2p.AttestationTarget{
		Slot:            b2.Slot,
		ParentRoot:      b2.ParentRoot,
		BeaconBlockRoot: b2Root[:],
	}

	// We give block D 2 votes.
	attestationTargets[5] = &pbp2p.AttestationTarget{
		Slot:            b3.Slot,
		ParentRoot:      b3.ParentRoot,
		BeaconBlockRoot: b3Root[:],
	}
	attestationTargets[6] = &pbp2p.AttestationTarget{
		Slot:            b3.Slot,
		ParentRoot:      b3.ParentRoot,
		BeaconBlockRoot: b3Root[:],
	}

	// We give block B 3 votes.
	attestationTargets[7] = &pbp2p.AttestationTarget{
		Slot:            b4.Slot,
		ParentRoot:      b4.ParentRoot,
		BeaconBlockRoot: b4Root[:],
	}
	attestationTargets[8] = &pbp2p.AttestationTarget{
		Slot:            b4.Slot,
		ParentRoot:      b4.ParentRoot,
		BeaconBlockRoot: b4Root[:],
	}
	attestationTargets[9] = &pbp2p.AttestationTarget{
		Slot:            b4.Slot,
		ParentRoot:      b4.ParentRoot,
		BeaconBlockRoot: b4Root[:],
	}

	// We give block E 1 vote.
	attestationTargets[10] = &pbp2p.AttestationTarget{
		Slot:            b5.Slot,
		ParentRoot:      b5.ParentRoot,
		BeaconBlockRoot: b5Root[:],
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
		if err := db.SaveBlockDeprecated(node.Block); err != nil {
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
		beaconDB: db,
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
