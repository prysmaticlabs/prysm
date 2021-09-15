package beaconclient

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestService_ReceiveBlocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		cfg:       &Config{BeaconClient: client},
		blockFeed: new(event.Feed),
	}
	stream := mock.NewMockBeaconChain_StreamBlocksClient(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	client.EXPECT().StreamBlocks(
		gomock.Any(),
		&ethpb.StreamBlocksRequest{},
	).Return(stream, nil)
	stream.EXPECT().Context().Return(ctx).AnyTimes()
	stream.EXPECT().Recv().Return(
		&ethpb.SignedBeaconBlock{},
		nil,
	).Do(func() {
		cancel()
	})
	bs.ReceiveBlocks(ctx)
}

func TestService_ReceiveAttestations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		cfg:                         &Config{BeaconClient: client},
		blockFeed:                   new(event.Feed),
		receivedAttestationsBuffer:  make(chan *ethpb.IndexedAttestation, 1),
		collectedAttestationsBuffer: make(chan []*ethpb.IndexedAttestation, 1),
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
		&emptypb.Empty{},
	).Return(stream, nil)
	stream.EXPECT().Context().Return(ctx).AnyTimes()
	stream.EXPECT().Recv().Return(
		att,
		nil,
	).Do(func() {
		cancel()
	})
	bs.ReceiveAttestations(ctx)
}

func TestService_ReceiveAttestations_Batched(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		cfg: &Config{
			BeaconClient: client,
			SlasherDB:    testDB.SetupSlasherDB(t, false),
		},
		blockFeed:                   new(event.Feed),
		attestationFeed:             new(event.Feed),
		receivedAttestationsBuffer:  make(chan *ethpb.IndexedAttestation, 1),
		collectedAttestationsBuffer: make(chan []*ethpb.IndexedAttestation, 1),
	}
	stream := mock.NewMockBeaconChain_StreamIndexedAttestationsClient(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	att := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Slot: 5,
			Target: &ethpb.Checkpoint{
				Epoch: 5,
				Root:  []byte("test root 1"),
			},
		},
		Signature: []byte{1, 2},
	}
	client.EXPECT().StreamIndexedAttestations(
		gomock.Any(),
		&emptypb.Empty{},
	).Return(stream, nil)
	stream.EXPECT().Context().Return(ctx).AnyTimes()
	stream.EXPECT().Recv().Return(
		att,
		nil,
	).Do(func() {
		// Let a slot pass for the ticker.
		time.Sleep(slots.DivideSlotBy(1))
		cancel()
	})

	go bs.ReceiveAttestations(ctx)
	bs.receivedAttestationsBuffer <- att
	att.Data.Target.Root = []byte("test root 2")
	bs.receivedAttestationsBuffer <- att
	att.Data.Target.Root = []byte("test root 3")
	bs.receivedAttestationsBuffer <- att
	atts := <-bs.collectedAttestationsBuffer
	require.Equal(t, 3, len(atts), "Unexpected number of attestations batched")
}
