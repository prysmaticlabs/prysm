package beaconclient

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/mock"
)

func TestService_ReceiveBlocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		beaconClient: client,
		blockFeed:    new(event.Feed),
	}
	stream := mock.NewMockBeaconChain_StreamBlocksClient(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	client.EXPECT().StreamBlocks(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(stream, nil)
	stream.EXPECT().Context().Return(ctx).AnyTimes()
	stream.EXPECT().Recv().Return(
		&ethpb.SignedBeaconBlock{},
		nil,
	).Do(func() {
		cancel()
	})
	bs.receiveBlocks(ctx)
}

func TestService_ReceiveAttestations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		beaconClient: client,
		blockFeed:    new(event.Feed),
	}
	stream := mock.NewMockBeaconChain_StreamIndexedAttestationsClient(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	att := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Slot: 5,
		},
	}
	client.EXPECT().StreamIndexedAttestations(
		gomock.Any(),
		&ptypes.Empty{},
	).Return(stream, nil)
	stream.EXPECT().Context().Return(ctx).AnyTimes()
	stream.EXPECT().Recv().Return(
		att,
		nil,
	).Do(func() {
		cancel()
	})
	bs.receiveAttestations(ctx)
}
