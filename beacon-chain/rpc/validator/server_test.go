package validator

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/go-ssz"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	blk "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	internal "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testing"
	mockRPC "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
	if err := db.SaveState(ctx, &pbp2p.BeaconState{}, [32]byte{}); err != nil {
		t.Fatal(err)
	}

	pubKey := []byte{'A'}
	if err := db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(pubKey), 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}

	Server := &Server{
		BeaconDB: db,
	}

	req := &pb.ValidatorIndexRequest{
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

	beaconState := &pbp2p.BeaconState{
		Slot:       0,
		Validators: []*ethpb.Validator{},
	}
	block := blk.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(ctx, block); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisRoot, err := ssz.SigningRoot(block)
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
	req := &pb.ValidatorActivationRequest{
		PublicKeys: [][]byte{[]byte("A")},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockChainStream := mockRPC.NewMockValidatorService_WaitForActivationServer(ctrl)
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

	priv1, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}
	priv2, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}
	pubKey1 := priv1.PublicKey().Marshal()[:]
	pubKey2 := priv2.PublicKey().Marshal()[:]

	if err := db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(pubKey1), 0); err != nil {
		t.Fatalf("Could not save validator index: %v", err)
	}
	if err := db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(pubKey2), 0); err != nil {
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
	genesisRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatalf("Could not get signing root %v", err)
	}
	depData := &ethpb.Deposit_Data{
		PublicKey:             pubKey1,
		WithdrawalCredentials: []byte("hey"),
	}
	signingRoot, err := ssz.SigningRoot(depData)
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
	depositCache.InsertDeposit(ctx, deposit, big.NewInt(10) /*blockNum*/, 0, depositTrie.Root())
	if err := db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(pubKey1), 0); err != nil {
		t.Fatalf("could not save validator index: %v", err)
	}
	if err := db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(pubKey2), 1); err != nil {
		t.Fatalf("could not save validator index: %v", err)
	}
	vs := &Server{
		BeaconDB:           db,
		Ctx:                context.Background(),
		CanonicalStateChan: make(chan *pbp2p.BeaconState, 1),
		ChainStartFetcher:  &mockPOW.POWChain{},
		BlockFetcher:       &mockPOW.POWChain{},
		Eth1InfoFetcher:    &mockPOW.POWChain{},
		DepositFetcher:     depositCache,
		HeadFetcher:        &mockChain.ChainService{State: beaconState, Root: genesisRoot[:]},
	}
	req := &pb.ValidatorActivationRequest{
		PublicKeys: [][]byte{pubKey1, pubKey2},
	}
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()
	mockChainStream := internal.NewMockValidatorService_WaitForActivationServer(ctrl)
	mockChainStream.EXPECT().Context().Return(context.Background())
	mockChainStream.EXPECT().Send(
		&pb.ValidatorActivationResponse{
			Statuses: []*pb.ValidatorActivationResponse_Status{
				{PublicKey: pubKey1,
					Status: &pb.ValidatorStatusResponse{
						Status:                 pb.ValidatorStatus_ACTIVE,
						Eth1DepositBlockNumber: 10,
						DepositInclusionSlot:   2218,
					},
				},
				{PublicKey: pubKey2,
					Status: &pb.ValidatorStatusResponse{
						ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
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
	Server := &Server{
		Ctx: ctx,
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		StateFeedListener: &mockChain.ChainService{},
		BeaconDB:          db,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockValidatorService_WaitForChainStartServer(ctrl)
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
	if err := db.SaveHeadBlockRoot(ctx, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, &pbp2p.BeaconState{Slot: 3}, headBlockRoot); err != nil {
		t.Fatal(err)
	}

	Server := &Server{
		Ctx: context.Background(),
		ChainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		StateFeedListener: &mockChain.ChainService{},
		BeaconDB:          db,
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockValidatorService_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&pb.ChainStartResponse{
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
	Server := &Server{
		Ctx:            context.Background(),
		ChainStartChan: make(chan time.Time, 1),
		ChainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		StateFeedListener: &mockChain.ChainService{},
		BeaconDB:          db,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockValidatorService_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&pb.ChainStartResponse{
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
	Server.ChainStartChan <- time.Unix(0, 0)
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Sending genesis time")
}

func BenchmarkAssignment(b *testing.B) {
	b.StopTimer()
	db := dbutil.SetupDB(b)
	defer dbutil.TeardownDB(b, db)
	ctx := context.Background()

	genesis := blk.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(ctx, genesis); err != nil {
		b.Fatalf("Could not save genesis block: %v", err)
	}
	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount * 4
	state, err := genesisState(validatorCount)
	if err != nil {
		b.Fatalf("Could not setup genesis state: %v", err)
	}
	genesisRoot, err := ssz.SigningRoot(genesis)
	if err != nil {
		b.Fatalf("Could not get signing root %v", err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, validatorCount)
	for i := 0; i < int(validatorCount); i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		copy(pubKeyBuf[:], []byte(strconv.Itoa(i)))
		wg.Add(1)
		go func(index int) {
			errs <- db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(pubKeyBuf), uint64(index))
			wg.Done()
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			b.Fatal(err)
		}
	}

	vs := &Server{
		BeaconDB:    db,
		HeadFetcher: &mockChain.ChainService{State: state, Root: genesisRoot[:]},
	}

	// Set up request for 100 public keys at a time
	pubKeys := make([][]byte, 100)
	for i := 0; i < len(pubKeys); i++ {
		buf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		copy(buf, []byte(strconv.Itoa(i)))
		pubKeys[i] = buf
	}

	req := &pb.AssignmentRequest{
		PublicKeys: pubKeys,
		EpochStart: 0,
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := vs.CommitteeAssignment(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

func genesisState(validators uint64) (*pbp2p.BeaconState, error) {
	genesisTime := time.Unix(0, 0).Unix()
	deposits := make([]*ethpb.Deposit, validators)
	for i := 0; i < len(deposits); i++ {
		var pubKey [96]byte
		copy(pubKey[:], []byte(strconv.Itoa(i)))
		depositData := &ethpb.Deposit_Data{
			PublicKey: pubKey[:],
			Amount:    params.BeaconConfig().MaxEffectiveBalance,
		}

		deposits[i] = &ethpb.Deposit{Data: depositData}
	}
	return state.GenesisBeaconState(deposits, uint64(genesisTime), &ethpb.Eth1Data{})
}
