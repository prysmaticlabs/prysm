package client

import (
	"bytes"
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	p2pmock "github.com/prysmaticlabs/prysm/shared/p2p/mock"
	"github.com/prysmaticlabs/prysm/validator/internal"
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

	m.broadcaster.EXPECT().Broadcast(
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	)

	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.StateRootResponse{
		StateRoot: []byte{'F'},
	}, nil /*err*/)

	validator.ProposeBlock(context.Background(), 55)
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
