package grpc_api

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	eventClient "github.com/prysmaticlabs/prysm/v5/api/client/event"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	mock2 "github.com/prysmaticlabs/prysm/v5/testing/mock"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestWaitForChainStart_StreamSetupFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	beaconNodeValidatorClient := mock2.NewMockBeaconNodeValidatorClient(ctrl)
	beaconNodeValidatorClient.EXPECT().WaitForChainStart(
		gomock.Any(),
		gomock.Any(),
	).Return(nil, errors.New("failed stream"))

	validatorClient := &grpcValidatorClient{beaconNodeValidatorClient, true}
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	want := "could not setup beacon chain ChainStart streaming client"
	assert.ErrorContains(t, want, err)
}

func TestStartEventStream(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	beaconNodeValidatorClient := mock2.NewMockBeaconNodeValidatorClient(ctrl)
	grpcClient := &grpcValidatorClient{beaconNodeValidatorClient, true}
	tests := []struct {
		name    string
		topics  []string
		prepare func()
		verify  func(t *testing.T, event *eventClient.Event)
	}{
		{
			name:   "Happy path Head topic",
			topics: []string{"head"},
			prepare: func() {
				stream := mock2.NewMockBeaconNodeValidator_StreamSlotsClient(ctrl)
				beaconNodeValidatorClient.EXPECT().StreamSlots(gomock.Any(),
					&eth.StreamSlotsRequest{VerifiedOnly: true}).Return(stream, nil)
				stream.EXPECT().Context().Return(ctx).AnyTimes()
				stream.EXPECT().Recv().Return(
					&eth.StreamSlotsResponse{Slot: 123},
					nil,
				).AnyTimes()
			},
			verify: func(t *testing.T, event *eventClient.Event) {
				require.Equal(t, event.EventType, eventClient.EventHead)
				head := structs.HeadEvent{}
				require.NoError(t, json.Unmarshal(event.Data, &head))
				require.Equal(t, head.Slot, "123")
			},
		},
		{
			name:   "no head produces error",
			topics: []string{"unsupportedTopic"},
			prepare: func() {
				stream := mock2.NewMockBeaconNodeValidator_StreamSlotsClient(ctrl)
				beaconNodeValidatorClient.EXPECT().StreamSlots(gomock.Any(),
					&eth.StreamSlotsRequest{VerifiedOnly: true}).Return(stream, nil)
				stream.EXPECT().Context().Return(ctx).AnyTimes()
				stream.EXPECT().Recv().Return(
					&eth.StreamSlotsResponse{Slot: 123},
					nil,
				).AnyTimes()
			},
			verify: func(t *testing.T, event *eventClient.Event) {
				require.Equal(t, event.EventType, eventClient.EventConnectionError)
			},
		},
		{
			name:   "Unsupported topics warning",
			topics: []string{"head", "unsupportedTopic"},
			prepare: func() {
				stream := mock2.NewMockBeaconNodeValidator_StreamSlotsClient(ctrl)
				beaconNodeValidatorClient.EXPECT().StreamSlots(gomock.Any(),
					&eth.StreamSlotsRequest{VerifiedOnly: true}).Return(stream, nil)
				stream.EXPECT().Context().Return(ctx).AnyTimes()
				stream.EXPECT().Recv().Return(
					&eth.StreamSlotsResponse{Slot: 123},
					nil,
				).AnyTimes()
			},
			verify: func(t *testing.T, event *eventClient.Event) {
				require.Equal(t, event.EventType, eventClient.EventHead)
				head := structs.HeadEvent{}
				require.NoError(t, json.Unmarshal(event.Data, &head))
				require.Equal(t, head.Slot, "123")
				assert.LogsContain(t, hook, "gRPC only supports the head topic")
			},
		},
		{
			name:    "No topics error",
			topics:  []string{},
			prepare: func() {},
			verify: func(t *testing.T, event *eventClient.Event) {
				require.Equal(t, event.EventType, eventClient.EventError)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eventsChannel := make(chan *eventClient.Event, 1) // Buffer to prevent blocking
			tc.prepare()                                      // Setup mock expectations

			go grpcClient.StartEventStream(ctx, tc.topics, eventsChannel)

			event := <-eventsChannel
			// Depending on what you're testing, you may need a timeout or a specific number of events to read
			time.AfterFunc(1*time.Second, cancel) // Prevents hanging forever
			tc.verify(t, event)
		})
	}
}
