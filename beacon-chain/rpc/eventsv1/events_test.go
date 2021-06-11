package eventsv1

import (
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStreamEvents_Preconditions(t *testing.T) {
	t.Run("no_topics_specified", func(t *testing.T) {
		srv := &Server{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStream := mock.NewMockEvents_StreamEventsClient(ctrl)
		err := srv.StreamEvents(&ethpb.StreamEventsRequest{Topics: nil}, mockStream)
		require.ErrorContains(t, "no topics specified", err)
	})
	t.Run("topic_not_allowed", func(t *testing.T) {
		srv := &Server{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStream := mock.NewMockEvents_StreamEventsClient(ctrl)
		err := srv.StreamEvents(&ethpb.StreamEventsRequest{Topics: []string{"foobar"}}, mockStream)
		require.ErrorContains(t, "topic foobar not allowed", err)
	})
}

func TestStreamEvents_BlockEvents(t *testing.T) {
	t.Run("received_block", func(t *testing.T) {

	})
}

//func TestServer_StreamAttestations_ContextCanceled(t *testing.T) {
//	ctx := context.Background()
//
//	ctx, cancel := context.WithCancel(ctx)
//	chainService := &chainMock.ChainService{}
//	server := &Server{
//		Ctx:                 ctx,
//		AttestationNotifier: chainService.OperationNotifier(),
//	}
//
//	exitRoutine := make(chan bool)
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//	mockStream := mock.NewMockBeaconChain_StreamAttestationsServer(ctrl)
//	mockStream.EXPECT().Context().Return(ctx)
//	go func(tt *testing.T) {
//		err := server.StreamAttestations(
//			&emptypb.Empty{},
//			mockStream,
//		)
//		assert.ErrorContains(tt, "Context canceled", err)
//		<-exitRoutine
//	}(t)
//	cancel()
//	exitRoutine <- true
//}
//
