package rpc

import (
	"context"
	"strings"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/go-ssz"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	mockRPC "github.com/prysmaticlabs/prysm/beacon-chain/rpc/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	dbt "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
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
		eth1InfoFetcher:   &mockPOW.POWChain{},
		stateFeedListener: &mockChain.ChainService{},
		beaconDB: db,
	}
	b := blocks.NewGenesisBlock([]byte{'A'})
	r, _ := ssz.SigningRoot(b)
	if err := beaconServer.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	if err := beaconServer.beaconDB.SaveHeadBlockRoot(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := beaconServer.beaconDB.SaveGenesisBlockRoot(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := beaconServer.beaconDB.SaveState(ctx, &pbp2p.BeaconState{}, r); err != nil {
		t.Fatal(err)
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
	beaconServer := &BeaconServer{
		ctx: context.Background(),
		chainStartFetcher: &mockPOW.POWChain{
			ChainFeed: new(event.Feed),
		},
		eth1InfoFetcher:   &mockPOW.POWChain{},
		stateFeedListener: &mockChain.ChainService{},
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
	hook := logTest.NewGlobal()
	beaconServer := &BeaconServer{
		ctx:            context.Background(),
		chainStartChan: make(chan time.Time, 1),
		chainStartFetcher: &mockPOW.FaultyMockPOWChain{
			ChainFeed: new(event.Feed),
		},
		eth1InfoFetcher:   &mockPOW.POWChain{},
		stateFeedListener: &mockChain.ChainService{},
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
