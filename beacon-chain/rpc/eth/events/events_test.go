package events

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/grpc-ecosystem/grpc-gateway/v2/proto/gateway"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v3/async/event"
	mockChain "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/mock"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestStreamEvents_Preconditions(t *testing.T) {
	t.Run("no_topics_specified", func(t *testing.T) {
		srv := &Server{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStream := mock.NewMockEvents_StreamEventsServer(ctrl)
		err := srv.StreamEvents(&ethpb.StreamEventsRequest{Topics: nil}, mockStream)
		require.ErrorContains(t, "No topics specified", err)
	})
	t.Run("topic_not_allowed", func(t *testing.T) {
		srv := &Server{}
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockStream := mock.NewMockEvents_StreamEventsServer(ctrl)
		err := srv.StreamEvents(&ethpb.StreamEventsRequest{Topics: []string{"foobar"}}, mockStream)
		require.ErrorContains(t, "Topic foobar not allowed", err)
	})
}

func TestStreamEvents_BlockEvents(t *testing.T) {
	t.Run(BlockTopic, func(t *testing.T) {
		ctx := context.Background()
		srv, ctrl, mockStream := setupServer(ctx, t)
		defer ctrl.Finish()

		wantedBlock := util.HydrateSignedBeaconBlock(&eth.SignedBeaconBlock{
			Block: &eth.BeaconBlock{
				Slot: 8,
			},
		})
		wantedBlockRoot, err := wantedBlock.HashTreeRoot()
		require.NoError(t, err)
		genericResponse, err := anypb.New(&ethpb.EventBlock{
			Slot:                8,
			Block:               wantedBlockRoot[:],
			ExecutionOptimistic: true,
		})
		require.NoError(t, err)
		wantedMessage := &gateway.EventSource{
			Event: BlockTopic,
			Data:  genericResponse,
		}
		wsb, err := blocks.NewSignedBeaconBlock(wantedBlock)
		require.NoError(t, err)
		assertFeedSendAndReceive(ctx, &assertFeedArgs{
			t:             t,
			srv:           srv,
			topics:        []string{BlockTopic},
			stream:        mockStream,
			shouldReceive: wantedMessage,
			itemToSend: &feed.Event{
				Type: blockfeed.ReceivedBlock,
				Data: &blockfeed.ReceivedBlockData{
					SignedBlock:  wsb,
					IsOptimistic: true,
				},
			},
			feed: srv.BlockNotifier.BlockFeed(),
		})
	})
}

func TestStreamEvents_OperationsEvents(t *testing.T) {
	t.Run("attestation_unaggregated", func(t *testing.T) {
		ctx := context.Background()
		srv, ctrl, mockStream := setupServer(ctx, t)
		defer ctrl.Finish()

		wantedAttV1alpha1 := util.HydrateAttestation(&eth.Attestation{
			Data: &eth.AttestationData{
				Slot: 8,
			},
		})
		wantedAtt := migration.V1Alpha1AttestationToV1(wantedAttV1alpha1)
		genericResponse, err := anypb.New(wantedAtt)
		require.NoError(t, err)

		wantedMessage := &gateway.EventSource{
			Event: AttestationTopic,
			Data:  genericResponse,
		}

		assertFeedSendAndReceive(ctx, &assertFeedArgs{
			t:             t,
			srv:           srv,
			topics:        []string{AttestationTopic},
			stream:        mockStream,
			shouldReceive: wantedMessage,
			itemToSend: &feed.Event{
				Type: operation.UnaggregatedAttReceived,
				Data: &operation.UnAggregatedAttReceivedData{
					Attestation: wantedAttV1alpha1,
				},
			},
			feed: srv.OperationNotifier.OperationFeed(),
		})
	})
	t.Run("attestation_aggregated", func(t *testing.T) {
		ctx := context.Background()
		srv, ctrl, mockStream := setupServer(ctx, t)
		defer ctrl.Finish()

		wantedAttV1alpha1 := &eth.AggregateAttestationAndProof{
			Aggregate: util.HydrateAttestation(&eth.Attestation{}),
		}
		wantedAtt := migration.V1Alpha1AggregateAttAndProofToV1(wantedAttV1alpha1)
		genericResponse, err := anypb.New(wantedAtt)
		require.NoError(t, err)

		wantedMessage := &gateway.EventSource{
			Event: AttestationTopic,
			Data:  genericResponse,
		}

		assertFeedSendAndReceive(ctx, &assertFeedArgs{
			t:             t,
			srv:           srv,
			topics:        []string{AttestationTopic},
			stream:        mockStream,
			shouldReceive: wantedMessage,
			itemToSend: &feed.Event{
				Type: operation.AggregatedAttReceived,
				Data: &operation.AggregatedAttReceivedData{
					Attestation: wantedAttV1alpha1,
				},
			},
			feed: srv.OperationNotifier.OperationFeed(),
		})
	})
	t.Run(VoluntaryExitTopic, func(t *testing.T) {
		ctx := context.Background()
		srv, ctrl, mockStream := setupServer(ctx, t)
		defer ctrl.Finish()

		wantedExitV1alpha1 := &eth.SignedVoluntaryExit{
			Exit: &eth.VoluntaryExit{
				Epoch:          1,
				ValidatorIndex: 1,
			},
			Signature: make([]byte, 96),
		}
		wantedExit := migration.V1Alpha1ExitToV1(wantedExitV1alpha1)
		genericResponse, err := anypb.New(wantedExit)
		require.NoError(t, err)

		wantedMessage := &gateway.EventSource{
			Event: VoluntaryExitTopic,
			Data:  genericResponse,
		}

		assertFeedSendAndReceive(ctx, &assertFeedArgs{
			t:             t,
			srv:           srv,
			topics:        []string{VoluntaryExitTopic},
			stream:        mockStream,
			shouldReceive: wantedMessage,
			itemToSend: &feed.Event{
				Type: operation.ExitReceived,
				Data: &operation.ExitReceivedData{
					Exit: wantedExitV1alpha1,
				},
			},
			feed: srv.OperationNotifier.OperationFeed(),
		})
	})
	t.Run(SyncCommitteeContributionTopic, func(t *testing.T) {
		ctx := context.Background()
		srv, ctrl, mockStream := setupServer(ctx, t)
		defer ctrl.Finish()

		wantedContributionV1alpha1 := &eth.SignedContributionAndProof{
			Message: &eth.ContributionAndProof{
				AggregatorIndex: 1,
				Contribution: &eth.SyncCommitteeContribution{
					Slot:              1,
					BlockRoot:         []byte("root"),
					SubcommitteeIndex: 1,
					AggregationBits:   bitfield.NewBitvector128(),
					Signature:         []byte("sig"),
				},
				SelectionProof: []byte("proof"),
			},
			Signature: []byte("sig"),
		}
		wantedContribution := migration.V1Alpha1SignedContributionAndProofToV2(wantedContributionV1alpha1)
		genericResponse, err := anypb.New(wantedContribution)
		require.NoError(t, err)

		wantedMessage := &gateway.EventSource{
			Event: SyncCommitteeContributionTopic,
			Data:  genericResponse,
		}

		assertFeedSendAndReceive(ctx, &assertFeedArgs{
			t:             t,
			srv:           srv,
			topics:        []string{SyncCommitteeContributionTopic},
			stream:        mockStream,
			shouldReceive: wantedMessage,
			itemToSend: &feed.Event{
				Type: operation.SyncCommitteeContributionReceived,
				Data: &operation.SyncCommitteeContributionReceivedData{
					Contribution: wantedContributionV1alpha1,
				},
			},
			feed: srv.OperationNotifier.OperationFeed(),
		})
	})
}

func TestStreamEvents_StateEvents(t *testing.T) {
	t.Run(HeadTopic, func(t *testing.T) {
		ctx := context.Background()
		srv, ctrl, mockStream := setupServer(ctx, t)
		defer ctrl.Finish()

		wantedHead := &ethpb.EventHead{
			Slot:                      8,
			Block:                     make([]byte, 32),
			State:                     make([]byte, 32),
			EpochTransition:           true,
			PreviousDutyDependentRoot: make([]byte, 32),
			CurrentDutyDependentRoot:  make([]byte, 32),
			ExecutionOptimistic:       true,
		}
		genericResponse, err := anypb.New(wantedHead)
		require.NoError(t, err)
		wantedMessage := &gateway.EventSource{
			Event: HeadTopic,
			Data:  genericResponse,
		}

		assertFeedSendAndReceive(ctx, &assertFeedArgs{
			t:             t,
			srv:           srv,
			topics:        []string{HeadTopic},
			stream:        mockStream,
			shouldReceive: wantedMessage,
			itemToSend: &feed.Event{
				Type: statefeed.NewHead,
				Data: wantedHead,
			},
			feed: srv.StateNotifier.StateFeed(),
		})
	})
	t.Run(FinalizedCheckpointTopic, func(t *testing.T) {
		ctx := context.Background()
		srv, ctrl, mockStream := setupServer(ctx, t)
		defer ctrl.Finish()

		wantedCheckpoint := &ethpb.EventFinalizedCheckpoint{
			Block:               make([]byte, 32),
			State:               make([]byte, 32),
			Epoch:               8,
			ExecutionOptimistic: true,
		}
		genericResponse, err := anypb.New(wantedCheckpoint)
		require.NoError(t, err)
		wantedMessage := &gateway.EventSource{
			Event: FinalizedCheckpointTopic,
			Data:  genericResponse,
		}

		assertFeedSendAndReceive(ctx, &assertFeedArgs{
			t:             t,
			srv:           srv,
			topics:        []string{FinalizedCheckpointTopic},
			stream:        mockStream,
			shouldReceive: wantedMessage,
			itemToSend: &feed.Event{
				Type: statefeed.FinalizedCheckpoint,
				Data: wantedCheckpoint,
			},
			feed: srv.StateNotifier.StateFeed(),
		})
	})
	t.Run(ChainReorgTopic, func(t *testing.T) {
		ctx := context.Background()
		srv, ctrl, mockStream := setupServer(ctx, t)
		defer ctrl.Finish()

		wantedReorg := &ethpb.EventChainReorg{
			Slot:                8,
			Depth:               1,
			OldHeadBlock:        make([]byte, 32),
			NewHeadBlock:        make([]byte, 32),
			OldHeadState:        make([]byte, 32),
			NewHeadState:        make([]byte, 32),
			Epoch:               0,
			ExecutionOptimistic: true,
		}
		genericResponse, err := anypb.New(wantedReorg)
		require.NoError(t, err)
		wantedMessage := &gateway.EventSource{
			Event: ChainReorgTopic,
			Data:  genericResponse,
		}

		assertFeedSendAndReceive(ctx, &assertFeedArgs{
			t:             t,
			srv:           srv,
			topics:        []string{ChainReorgTopic},
			stream:        mockStream,
			shouldReceive: wantedMessage,
			itemToSend: &feed.Event{
				Type: statefeed.Reorg,
				Data: wantedReorg,
			},
			feed: srv.StateNotifier.StateFeed(),
		})
	})
}

func TestStreamEvents_CommaSeparatedTopics(t *testing.T) {
	ctx := context.Background()
	srv, ctrl, mockStream := setupServer(ctx, t)
	defer ctrl.Finish()

	wantedHead := &ethpb.EventHead{
		Slot:                      8,
		Block:                     make([]byte, 32),
		State:                     make([]byte, 32),
		EpochTransition:           true,
		PreviousDutyDependentRoot: make([]byte, 32),
		CurrentDutyDependentRoot:  make([]byte, 32),
	}
	headGenericResponse, err := anypb.New(wantedHead)
	require.NoError(t, err)
	wantedHeadMessage := &gateway.EventSource{
		Event: HeadTopic,
		Data:  headGenericResponse,
	}

	assertFeedSendAndReceive(ctx, &assertFeedArgs{
		t:             t,
		srv:           srv,
		topics:        []string{HeadTopic + "," + FinalizedCheckpointTopic},
		stream:        mockStream,
		shouldReceive: wantedHeadMessage,
		itemToSend: &feed.Event{
			Type: statefeed.NewHead,
			Data: wantedHead,
		},
		feed: srv.StateNotifier.StateFeed(),
	})

	wantedCheckpoint := &ethpb.EventFinalizedCheckpoint{
		Block: make([]byte, 32),
		State: make([]byte, 32),
		Epoch: 8,
	}
	checkpointGenericResponse, err := anypb.New(wantedCheckpoint)
	require.NoError(t, err)
	wantedCheckpointMessage := &gateway.EventSource{
		Event: FinalizedCheckpointTopic,
		Data:  checkpointGenericResponse,
	}

	assertFeedSendAndReceive(ctx, &assertFeedArgs{
		t:             t,
		srv:           srv,
		topics:        []string{HeadTopic + "," + FinalizedCheckpointTopic},
		stream:        mockStream,
		shouldReceive: wantedCheckpointMessage,
		itemToSend: &feed.Event{
			Type: statefeed.FinalizedCheckpoint,
			Data: wantedCheckpoint,
		},
		feed: srv.StateNotifier.StateFeed(),
	})
}

func setupServer(ctx context.Context, t testing.TB) (*Server, *gomock.Controller, *mock.MockEvents_StreamEventsServer) {
	srv := &Server{
		BlockNotifier:     &mockChain.MockBlockNotifier{},
		StateNotifier:     &mockChain.MockStateNotifier{},
		OperationNotifier: &mockChain.MockOperationNotifier{},
		Ctx:               ctx,
	}
	ctrl := gomock.NewController(t)
	mockStream := mock.NewMockEvents_StreamEventsServer(ctrl)
	return srv, ctrl, mockStream
}

type assertFeedArgs struct {
	t             *testing.T
	topics        []string
	srv           *Server
	stream        *mock.MockEvents_StreamEventsServer
	shouldReceive interface{}
	itemToSend    *feed.Event
	feed          *event.Feed
}

func assertFeedSendAndReceive(ctx context.Context, args *assertFeedArgs) {
	exitRoutine := make(chan bool)
	defer close(exitRoutine)
	args.stream.EXPECT().Send(args.shouldReceive).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	args.stream.EXPECT().Context().Return(ctx).AnyTimes()

	req := &ethpb.StreamEventsRequest{Topics: args.topics}
	go func(tt *testing.T) {
		assert.NoError(tt, args.srv.StreamEvents(req, args.stream), "Could not call RPC method")
	}(args.t)
	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = args.feed.Send(args.itemToSend)
	}
	<-exitRoutine
}
