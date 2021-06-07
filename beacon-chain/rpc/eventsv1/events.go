package eventsv1

import (
	gwpb "github.com/grpc-ecosystem/grpc-gateway/v2/proto/gateway"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/proto/migration"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var casesHandled = map[string]bool{
	"head":                 true,
	"block":                true,
	"attestation":          true,
	"voluntary_exit":       true,
	"finalized_checkpoint": true,
	"chain_reorg":          true,
}

// StreamEvents allows requesting all events from a set of topics defined in the eth2.0-apis standard.
// The topics supported include block events, attestations, chain reorgs, voluntary exits,
// chain finality, and more.
func (s *Server) StreamEvents(
	req *ethpb.StreamEventsRequest, stream ethpb.Events_StreamEventsServer,
) error {
	// Check if the topics in the request are valid.
	for _, topic := range req.Topics {
		if _, ok := casesHandled[topic]; !ok {
			return status.Errorf(codes.InvalidArgument, "topic %s not allowed for event subscriptions", topic)
		}
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
			return s.handleBlockEvents(stream, event)
		case event := <-opsChan:
			return s.handleBlockOperationEvents(stream, event)
		case event := <-stateChan:
			return s.handleStateEvents(stream, event)
		case <-s.Ctx.Done():
			return errors.New("context canceled")
		case <-stream.Context().Done():
			return errors.New("context canceled")
		}
	}
}

func (s *Server) handleBlockEvents(stream ethpb.Events_StreamEventsServer, event *feed.Event) error {
	switch event.Type {
	// TODO: Handle new head.
	case blockfeed.ReceivedBlock:
		blkData, ok := event.Data.(*blockfeed.ReceivedBlockData)
		if !ok {
			return nil
		}
		v1Data, err := migration.BlockIfaceToV1Blockheader(blkData.SignedBlock)
		if err != nil {
			return err
		}
		return s.streamData(stream, "block", v1Data)
	default:
		return nil
	}
}

func (s *Server) handleBlockOperationEvents(stream ethpb.Events_StreamEventsServer, event *feed.Event) error {
	switch event.Type {
	case operation.AggregatedAttReceived:
		attData, ok := event.Data.(*operation.AggregatedAttReceivedData)
		if !ok {
			return nil
		}
		v1Data := migration.V1Alpha1AggregateAttAndProofToV1(attData.Attestation)
		return s.streamData(stream, "attestation", v1Data)
	case operation.UnaggregatedAttReceived:
		attData, ok := event.Data.(*operation.UnAggregatedAttReceivedData)
		if !ok {
			return nil
		}
		v1Data := migration.V1Alpha1AttestationToV1(attData.Attestation)
		return s.streamData(stream, "attestation", v1Data)
	case operation.ExitReceived:
		exitData, ok := event.Data.(*operation.ExitReceivedData)
		if !ok {
			return nil
		}
		return s.streamData(stream, "voluntary_exit", exitData.Exit)
	default:
		return nil
	}
}

func (s *Server) handleStateEvents(stream ethpb.Events_StreamEventsServer, event *feed.Event) error {
	switch event.Type {
	// TODO: Handle reorg.
	case statefeed.Reorg:
		return nil
	default:
		return nil
	}
}

func (s *Server) streamData(stream ethpb.Events_StreamEventsServer, name string, data proto.Message) error {
	returnData, err := anypb.New(data)
	if err != nil {
		log.WithError(err).Error("Could not parse request from pb")
		return err
	}
	return stream.Send(&gwpb.EventSource{
		Event: name,
		Data:  returnData,
	})
}
