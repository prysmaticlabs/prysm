package validator

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	blk "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	internal "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testing"
	mockRPC "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	// Use minimal config to reduce test setup time.
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
}

func TestValidatorIndex_OK(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	st, _ := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{})
	if err := db.SaveState(ctx, st.Copy(), [32]byte{}); err != nil {
		t.Fatal(err)
	}

	pubKey := pubKey(1)
	if err := db.SaveValidatorIndex(ctx, pubKey, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	Server := &Server{
		BeaconDB: db,
	}

	req := &ethpb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	if _, err := Server.ValidatorIndex(context.Background(), req); err != nil {
		t.Errorf("Could not get validator index: %v", err)
	}
}

func TestWaitForActivation_ContextClosed(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	beaconState, _ := stateTrie.InitializeFromProto(&pbp2p.BeaconState{
		Slot:       0,
		Validators: []*ethpb.Validator{},
	})
	block := blk.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := ssz.HashTreeRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	vs := &Server{
		BeaconDB:           db,
		Ctx:                ctx,
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		DepositFetcher:     depositcache.NewDepositCache(),
		HeadFetcher:        &mockChain.ChainService{State: beaconState, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorActivationRequest{
		PublicKeys: [][]byte{pubKey(1)},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockChainStream := mockRPC.NewMockBeaconNodeValidator_WaitForActivationServer(ctrl)
	mockChainStream.EXPECT().Context().Return(context.Background())
	mockChainStream.EXPECT().Send(gomock.Any()).Return(nil)
	mockChainStream.EXPECT().Context().Return(context.Background())
	exitRoutine := make(chan bool)
	go func(tt *testing.T) {
		want := "context canceled"
		if err := vs.WaitForActivation(req, mockChainStream); !strings.Contains(err.Error(), want) {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForActivation_ValidatorOriginallyExists(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	// This test breaks if it doesnt use mainnet config
	params.OverrideBeaconConfig(params.MainnetConfig())
	defer params.OverrideBeaconConfig(params.MinimalSpecConfig())
	ctx := context.Background()

	priv1 := bls.RandKey()
	priv2 := bls.RandKey()

	pubKey1 := priv1.PublicKey().Marshal()[:]
	pubKey2 := priv2.PublicKey().Marshal()[:]

	if err := db.SaveValidatorIndex(ctx, pubKey1, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}
	if err := db.SaveValidatorIndex(ctx, pubKey2, 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	beaconState := &pbp2p.BeaconState{
		Slot: 4000,
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch: 0,
				ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
				PublicKey:       pubKey1,
			},
		},
	}
	block := blk.NewGenesisBlock([]byte{})
	genesisRoot, err := ssz.HashTreeRoot(block.Block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey1,
		WithdrawalCredentials: []byte("hey"),
	}
	signingRoot, err := ssz.HashTreeRoot(depData)
	if err != nil {
		t.Error(err)
	}
	domain := bls.ComputeDomain(params.BeaconConfig().DomainDeposit)
	depData.Signature = priv1.Sign(signingRoot[:], domain).Marshal()[:]

	deposit := &ethpb.Deposit{
		Data: depData,
	}
	depositTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(fmt.Errorf("could not setup deposit trie: %v", err))
	}
	depositCache := depositcache.NewDepositCache()
	depositCache.InsertDeposit(ctx, deposit, 10 /*blockNum*/, 0, depositTrie.Root())
	if err := db.SaveValidatorIndex(ctx, pubKey1, 0); err != nil {
		t.Fatalf("could not save validator index: %v", err)
	}
	if err := db.SaveValidatorIndex(ctx, pubKey2, 1); err != nil {
		t.Fatalf("could not save validator index: %v", err)
	}
	trie, err := stateTrie.InitializeFromProtoUnsafe(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	vs := &Server{
		BeaconDB:           db,
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		DepositFetcher:     depositCache,
		HeadFetcher:        &mockChain.ChainService{State: trie, Root: genesisRoot[:]},
	}
	req := &ethpb.ValidatorActivationRequest{
		PublicKeys: [][]byte{pubKey1, pubKey2},
	}
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()
	mockChainStream := internal.NewMockBeaconNodeValidator_WaitForActivationServer(ctrl)
	mockChainStream.EXPECT().Context().Return(context.Background())
	mockChainStream.EXPECT().Send(
		&ethpb.ValidatorActivationResponse{
			Statuses: []*ethpb.ValidatorActivationResponse_Status{
				{PublicKey: pubKey1,
					Status: &ethpb.ValidatorStatusResponse{
						Status:                 ethpb.ValidatorStatus_ACTIVE,
						Eth1DepositBlockNumber: 10,
						DepositInclusionSlot:   2218,
					},
				},
				{PublicKey: pubKey2,
					Status: &ethpb.ValidatorStatusResponse{
						ActivationEpoch: int64(params.BeaconConfig().FarFutureEpoch),
					},
				},
			},
		},
	).Return(nil)

	if err := vs.WaitForActivation(req, mockChainStream); err != nil {
		t.Fatalf("Could not setup wait for activation stream: %v", err)
	}
}

func TestWaitForChainStart_ContextClosed(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()

	ctx, cancel := context.WithCancel(context.Background())
	chainService := &mockChain.ChainService{}
	Server := &Server{
		Ctx: ctx,
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		StateNotifier: chainService.StateNotifier(),
		BeaconDB:      db,
		HeadFetcher:   chainService,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	go func(tt *testing.T) {
		if err := Server.WaitForChainStart(&ptypes.Empty{}, mockStream); !strings.Contains(err.Error(), "Context canceled") {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForChainStart_AlreadyStarted(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	headBlockRoot := [32]byte{0x01, 0x02}
	trie, err := stateTrie.InitializeFromProtoUnsafe(&pbp2p.BeaconState{Slot: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, trie, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, headBlockRoot); err != nil {
		t.Fatal(err)
	}

	chainService := &mockChain.ChainService{State: trie}
	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		BeaconDB:      db,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&ethpb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	if err := Server.WaitForChainStart(&ptypes.Empty{}, mockStream); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
}

func TestWaitForChainStart_NotStartedThenLogFired(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)

	hook := logTest.NewGlobal()
	chainService := &mockChain.ChainService{}
	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		BeaconDB:      db,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconNodeValidator_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&ethpb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	go func(tt *testing.T) {
		if err := Server.WaitForChainStart(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = Server.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.ChainStarted,
			Data: &statefeed.ChainStartedData{
				StartTime: time.Unix(0, 0),
			},
		})
	}

	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Sending genesis time")
}
