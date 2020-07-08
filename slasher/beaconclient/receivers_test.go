package beaconclient

import (
	"context"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
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
	bs.ReceiveBlocks(ctx)
}

func TestService_ReceiveAttestations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		beaconClient:                client,
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
		&ptypes.Empty{},
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
		beaconClient:                client,
		blockFeed:                   new(event.Feed),
		slasherDB:                   testDB.SetupSlasherDB(t, false),
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
		&ptypes.Empty{},
	).Return(stream, nil)
	stream.EXPECT().Context().Return(ctx).AnyTimes()
	stream.EXPECT().Recv().Return(
		att,
		nil,
	).Do(func() {
		// Let a slot pass for the ticker.
		time.Sleep(slotutil.DivideSlotBy(1))
		cancel()
	})

	go bs.ReceiveAttestations(ctx)
	bs.receivedAttestationsBuffer <- att
	att.Data.Target.Root = []byte("test root 2")
	bs.receivedAttestationsBuffer <- att
	att.Data.Target.Root = []byte("test root 3")
	bs.receivedAttestationsBuffer <- att
	atts := <-bs.collectedAttestationsBuffer
	if len(atts) != 3 {
		t.Fatalf("Expected %d received attestations to be batched", len(atts))
	}
}
