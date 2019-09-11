package rpc

import (
	"context"
	"strings"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbt "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	mockRPC "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testing"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestWaitForChainStart_ContextClosed(t *testing.T) {
	db := dbt.SetupDB(t)
	defer dbt.TeardownDB(t, db)
	ctx := context.Background()

	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx: ctx,
		chainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		stateFeedListener: &mockChain.ChainService{},
		beaconDB:          db,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconService_WaitForChainStartServer(ctrl)
	go func(tt *testing.T) {
		if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); !strings.Contains(err.Error(), "context closed") {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForChainStart_AlreadyStarted(t *testing.T) {
	db := dbt.SetupDB(t)
	defer dbt.TeardownDB(t, db)
	ctx := context.Background()
	headBlockRoot := [32]byte{0x01, 0x02}
	if err := db.SaveHeadBlockRoot(ctx, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, &ethereum_beacon_p2p_v1.BeaconState{Slot: 3}, headBlockRoot); err != nil {
		t.Fatal(err)
	}

	beaconServer := &BeaconServer{
		ctx: context.Background(),
		chainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		stateFeedListener: &mockChain.ChainService{},
		beaconDB:          db,
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconService_WaitForChainStartServer(ctrl)
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
	db := dbt.SetupDB(t)
	defer dbt.TeardownDB(t, db)

	hook := logTest.NewGlobal()
	beaconServer := &BeaconServer{
		ctx:            context.Background(),
		chainStartChan: make(chan time.Time, 1),
		chainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		stateFeedListener: &mockChain.ChainService{},
		beaconDB:          db,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mockRPC.NewMockBeaconService_WaitForChainStartServer(ctrl)
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
