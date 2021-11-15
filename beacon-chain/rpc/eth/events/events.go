package events

import (
	gwpb "github.com/grpc-ecosystem/grpc-gateway/v2/proto/gateway"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// HeadTopic represents a new chain head event topic.
	HeadTopic = "head"
	// BlockTopic represents a new produced block event topic.
	BlockTopic = "block"
	// AttestationTopic represents a new submitted attestation event topic.
	AttestationTopic = "attestation"
	// VoluntaryExitTopic represents a new performed voluntary exit event topic.
	VoluntaryExitTopic = "voluntary_exit"
	// FinalizedCheckpointTopic represents a new finalized checkpoint event topic.
	FinalizedCheckpointTopic = "finalized_checkpoint"
	// ChainReorgTopic represents a chain reorganization event topic.
	ChainReorgTopic = "chain_reorg"
	// SyncCommitteeContributionTopic represents a new sync committee contribution event topic.
	SyncCommitteeContributionTopic = "contribution_and_proof"
)

var casesHandled = map[string]bool{
	HeadTopic:                      true,
	BlockTopic:                     true,
	AttestationTopic:               true,
	VoluntaryExitTopic:             true,
	FinalizedCheckpointTopic:       true,
	ChainReorgTopic:                true,
	SyncCommitteeContributionTopic: true,
}

// StreamEvents allows requesting all events from a set of topics defined in the Ethereum consensus API standard.
// The topics supported include block events, attestations, chain reorgs, voluntary exits,
// chain finality, and more.
func (s *Server) StreamEvents(
	req *ethpb.StreamEventsRequest, stream ethpbservice.Events_StreamEventsServer,
) error {
	if req == nil || len(req.Topics) == 0 {
		return status.Error(codes.InvalidArgument, "No topics specified to subscribe to")
	}
	// Check if the topics in the request are valid.
	requestedTopics := make(map[string]bool)
	for _, topic := range req.Topics {
		if _, ok := casesHandled[topic]; !ok {
			return status.Errorf(codes.InvalidArgument, "Topic %s not allowed for event subscriptions", topic)
		}
		requestedTopics[topic] = true
	}

	// Subscribe to event feeds from information received in the beacon node runtime.
	blockChan := make(chan *feed.Event, 1)
	blockSub := s.BlockNotifier.BlockFeed().Subscribe(blockChan)

	opsChan := make(chan *feed.Event, 1)
	opsSub := s.OperationNotifier.OperationFeed().Subscribe(opsChan)

	stateChan := make(chan *feed.Event, 1)
	stateSub := s.StateNotifier.StateFeed().Subscribe(stateChan)

	defer blockSub.Unsubscribe()
	defer opsSub.Unsubscribe()
	defer stateSub.Unsubscribe()

	// Handle each event received and context cancelation.
	for {
		select {
		case event := <-blockChan:
			if err := handleBlockEvents(stream, requestedTopics, event); err != nil {
				return status.Errorf(codes.Internal, "Could not handle block event: %v", err)
			}
		case event := <-opsChan:
			if err := handleBlockOperationEvents(stream, requestedTopics, event); err != nil {
				return status.Errorf(codes.Internal, "Could not handle block operations event: %v", err)
			}
		case event := <-stateChan:
			if err := handleStateEvents(stream, requestedTopics, event); err != nil {
				return status.Errorf(codes.Internal, "Could not handle state event: %v", err)
			}
		case <-s.Ctx.Done():
			return status.Errorf(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Errorf(codes.Canceled, "Context canceled")
		}
	}
}

func handleBlockEvents(
	stream ethpbservice.Events_StreamEventsServer, requestedTopics map[string]bool, event *feed.Event,
) error {
	switch event.Type {
	case blockfeed.ReceivedBlock:
		if _, ok := requestedTopics[BlockTopic]; !ok {
			return nil
		}
		blkData, ok := event.Data.(*blockfeed.ReceivedBlockData)
		if !ok {
			return nil
		}
		v1Data, err := migration.BlockIfaceToV1BlockHeader(blkData.SignedBlock)
		if err != nil {
			return err
		}
		item, err := v1Data.HashTreeRoot()
		if err != nil {
			return errors.Wrap(err, "could not hash tree root block")
		}
		eventBlock := &ethpb.EventBlock{
			Slot:  v1Data.Message.Slot,
			Block: item[:],
		}
		return streamData(stream, BlockTopic, eventBlock)
	default:
		return nil
	}
}

func handleBlockOperationEvents(
	stream ethpbservice.Events_StreamEventsServer, requestedTopics map[string]bool, event *feed.Event,
) error {
	switch event.Type {
	case operation.AggregatedAttReceived:
		if _, ok := requestedTopics[AttestationTopic]; !ok {
			return nil
		}
		attData, ok := event.Data.(*operation.AggregatedAttReceivedData)
		if !ok {
			return nil
		}
		v1Data := migration.V1Alpha1AggregateAttAndProofToV1(attData.Attestation)
		return streamData(stream, AttestationTopic, v1Data)
	case operation.UnaggregatedAttReceived:
		if _, ok := requestedTopics[AttestationTopic]; !ok {
			return nil
		}
		attData, ok := event.Data.(*operation.UnAggregatedAttReceivedData)
		if !ok {
			return nil
		}
		v1Data := migration.V1Alpha1AttestationToV1(attData.Attestation)
		return streamData(stream, AttestationTopic, v1Data)
	case operation.ExitReceived:
		if _, ok := requestedTopics[VoluntaryExitTopic]; !ok {
			return nil
		}
		exitData, ok := event.Data.(*operation.ExitReceivedData)
		if !ok {
			return nil
		}
		v1Data := migration.V1Alpha1ExitToV1(exitData.Exit)
		return streamData(stream, VoluntaryExitTopic, v1Data)
	case operation.SyncCommitteeContributionReceived:
		if _, ok := requestedTopics[SyncCommitteeContributionTopic]; !ok {
			return nil
		}
		contributionData, ok := event.Data.(*operation.SyncCommitteeContributionReceivedData)
		if !ok {
			return nil
		}
		v2Data := migration.V1Alpha1SignedContributionAndProofToV2(contributionData.Contribution)
		return streamData(stream, SyncCommitteeContributionTopic, v2Data)
	default:
		return nil
	}
}

func handleStateEvents(
	stream ethpbservice.Events_StreamEventsServer, requestedTopics map[string]bool, event *feed.Event,
) error {
	switch event.Type {
	case statefeed.NewHead:
		if _, ok := requestedTopics[HeadTopic]; !ok {
			return nil
		}
		head, ok := event.Data.(*ethpb.EventHead)
		if !ok {
			return nil
		}
		return streamData(stream, HeadTopic, head)
	case statefeed.FinalizedCheckpoint:
		if _, ok := requestedTopics[FinalizedCheckpointTopic]; !ok {
			return nil
		}
		finalizedCheckpoint, ok := event.Data.(*ethpb.EventFinalizedCheckpoint)
		if !ok {
			return nil
		}
		return streamData(stream, FinalizedCheckpointTopic, finalizedCheckpoint)
	case statefeed.Reorg:
		if _, ok := requestedTopics[ChainReorgTopic]; !ok {
			return nil
		}
		reorg, ok := event.Data.(*ethpb.EventChainReorg)
		if !ok {
			return nil
		}
		return streamData(stream, ChainReorgTopic, reorg)
	default:
		return nil
	}
}

func streamData(stream ethpbservice.Events_StreamEventsServer, name string, data proto.Message) error {
	returnData, err := anypb.New(data)
	if err != nil {
		return err
	}
	return stream.Send(&gwpb.EventSource{
		Event: name,
		Data:  returnData,
	})
}
