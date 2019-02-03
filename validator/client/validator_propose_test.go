package client

import (
	"bytes"
	"context"
	"errors"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	p2pmock "github.com/prysmaticlabs/prysm/shared/p2p/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/internal"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mocks struct {
	broadcaster    *p2pmock.MockBroadcaster
	proposerClient *internal.MockProposerServiceClient
	beaconClient   *internal.MockBeaconServiceClient
}

func setup(t *testing.T) (*validator, *mocks, func()) {
	ctrl := gomock.NewController(t)
	m := &mocks{
		broadcaster:    p2pmock.NewMockBroadcaster(ctrl),
		proposerClient: internal.NewMockProposerServiceClient(ctrl),
		beaconClient:   internal.NewMockBeaconServiceClient(ctrl),
	}

	validator := &validator{
		p2p:             m.broadcaster,
		attestationPool: &fakeAttestationPool{},
		proposerClient:  m.proposerClient,
		beaconClient:    m.beaconClient,
	}

	return validator, m, ctrl.Finish
}

func TestProposeBlock_LogsCanonicalHeadFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(nil /*beaconBlock*/, errors.New("something bad happened"))

	validator.ProposeBlock(context.Background(), 55)

	testutil.AssertLogsContain(t, hook, "something bad happened")
}

func TestProposeBlock_PendingDepositsFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(nil /*response*/, errors.New("something bad happened"))

	validator.ProposeBlock(context.Background(), 55)

	testutil.AssertLogsContain(t, hook, "something bad happened")
}

func TestProposeBlock_UsePendingDeposits(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{
		PendingDeposits: []*pbp2p.Deposit{
			&pbp2p.Deposit{DepositData: []byte{'D', 'A', 'T', 'A'}},
		},
	}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{}, nil /*err*/)

	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.StateRootResponse{
		StateRoot: []byte{'F'},
	}, nil /*err*/)

	var broadcastedBlock *pbp2p.BeaconBlock
	m.broadcaster.EXPECT().Broadcast(
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Do(func(blk *pbp2p.BeaconBlock) {
		broadcastedBlock = blk
	})

	validator.ProposeBlock(context.Background(), 55)

	if !bytes.Equal(broadcastedBlock.Body.Deposits[0].DepositData, []byte{'D', 'A', 'T', 'A'}) {
		t.Errorf("Unexpected deposit data: %v", broadcastedBlock.Body.Deposits)
	}
}

func TestProposeBlock_Eth1DataFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(nil /*response*/, errors.New("something bad happened"))

	validator.ProposeBlock(context.Background(), 55)

	testutil.AssertLogsContain(t, hook, "something bad happened")
}

func TestProposeBlock_UsesEth1Data(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{
		Eth1Data: &pbp2p.Eth1Data{BlockHash32: []byte{'B', 'L', 'O', 'C', 'K'}},
	}, nil /*err*/)

	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.StateRootResponse{
		StateRoot: []byte{'F'},
	}, nil /*err*/)

	var broadcastedBlock *pbp2p.BeaconBlock
	m.broadcaster.EXPECT().Broadcast(
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Do(func(blk *pbp2p.BeaconBlock) {
		broadcastedBlock = blk
	})

	validator.ProposeBlock(context.Background(), 55)

	if !bytes.Equal(broadcastedBlock.Eth1Data.BlockHash32, []byte{'B', 'L', 'O', 'C', 'K'}) {
		t.Errorf("Unexpected ETH1 data: %v", broadcastedBlock.Eth1Data)
	}
}

func TestProposeBlock_UsesComputedState(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{}, nil /*err*/)

	var broadcastedBlock *pbp2p.BeaconBlock
	m.broadcaster.EXPECT().Broadcast(
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Do(func(blk *pbp2p.BeaconBlock) {
		broadcastedBlock = blk
	})

	computedStateRoot := []byte{'T', 'E', 'S', 'T'}
	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(
		&pb.StateRootResponse{
			StateRoot: computedStateRoot,
		},
		nil, // err
	)

	validator.ProposeBlock(context.Background(), 55)

	if !bytes.Equal(broadcastedBlock.StateRootHash32, computedStateRoot) {
		t.Errorf("Unexpected state root hash. want=%#x got=%#x", computedStateRoot, broadcastedBlock.StateRootHash32)
	}
}

func TestProposeBlock_BroadcastsABlock(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{}, nil /*err*/)

	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.StateRootResponse{
		StateRoot: []byte{'F'},
	}, nil /*err*/)

	m.broadcaster.EXPECT().Broadcast(
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	)

	validator.ProposeBlock(context.Background(), 55)
}
