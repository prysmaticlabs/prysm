package eventsv1

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/grpc-ecosystem/grpc-gateway/v2/proto/gateway"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb_v1alpha1 "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestStreamEvents_Preconditions(t *testing.T) {
	t.Run("no_topics_specified", func(t *testing.T) {
		srv := &Server{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStream := mock.NewMockEvents_StreamEventsServer(ctrl)
		err := srv.StreamEvents(&ethpb.StreamEventsRequest{Topics: nil}, mockStream)
		require.ErrorContains(t, "no topics specified", err)
	})
	t.Run("topic_not_allowed", func(t *testing.T) {
		srv := &Server{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStream := mock.NewMockEvents_StreamEventsServer(ctrl)
		err := srv.StreamEvents(&ethpb.StreamEventsRequest{Topics: []string{"foobar"}}, mockStream)
		require.ErrorContains(t, "topic foobar not allowed", err)
	})
}

func TestStreamEvents_BlockEvents(t *testing.T) {
	t.Run("received_block", func(t *testing.T) {
		ctx := context.Background()
		srv := &Server{
			BlockNotifier:     &mockChain.MockBlockNotifier{},
			StateNotifier:     &mockChain.MockStateNotifier{},
			OperationNotifier: &mockChain.MockOperationNotifier{},
			Ctx:               ctx,
		}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStream := mock.NewMockEvents_StreamEventsServer(ctrl)

		wantedBlock := testutil.HydrateSignedBeaconBlock(&ethpb_v1alpha1.SignedBeaconBlock{
			Block: &ethpb_v1alpha1.BeaconBlock{
				Slot: 8,
			},
		})
		wantedBlockRoot, err := wantedBlock.HashTreeRoot()
		require.NoError(t, err)
		genericResponse, err := anypb.New(&ethpb.EventBlock{
			Slot:  8,
			Block: wantedBlockRoot[:],
		})
		require.NoError(t, err)

		want := &gateway.EventSource{
			Event: "block",
			Data:  genericResponse,
		}

		exitRoutine := make(chan bool)
		defer close(exitRoutine)
		mockStream.EXPECT().Send(want).Do(func(arg0 interface{}) {
			exitRoutine <- true
		})
		mockStream.EXPECT().Context().Return(ctx).AnyTimes()

		req := &ethpb.StreamEventsRequest{Topics: []string{"block"}}
		go func(tt *testing.T) {
			assert.NoError(tt, srv.StreamEvents(req, mockStream), "Could not call RPC method")
		}(t)
		// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
		for sent := 0; sent == 0; {
			sent = srv.BlockNotifier.BlockFeed().Send(&feed.Event{
				Type: blockfeed.ReceivedBlock,
				Data: &blockfeed.ReceivedBlockData{
					SignedBlock: interfaces.WrappedPhase0SignedBeaconBlock(wantedBlock),
				},
			})
		}
		<-exitRoutine
	})
}
