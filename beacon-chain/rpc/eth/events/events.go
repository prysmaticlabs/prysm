package events

import (
	"strings"

	gwpb "github.com/grpc-ecosystem/grpc-gateway/v2/proto/gateway"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpbservice "github.com/prysmaticlabs/prysm/v4/proto/eth/service"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
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
	// BLSToExecutionChangeTopic represents a new received BLS to execution change event topic.
	BLSToExecutionChangeTopic = "bls_to_execution_change"
	// PayloadAttributesTopic represents a new payload attributes for execution payload building event topic.
	PayloadAttributesTopic = "payload_attributes"
	// BlobSidecarTopic represents a new blob sidecar event topic
	BlobSidecarTopic = "blob_sidecar"
)

var casesHandled = map[string]bool{
	HeadTopic:                      true,
	BlockTopic:                     true,
	AttestationTopic:               true,
	VoluntaryExitTopic:             true,
	FinalizedCheckpointTopic:       true,
	ChainReorgTopic:                true,
	SyncCommitteeContributionTopic: true,
	BLSToExecutionChangeTopic:      true,
	PayloadAttributesTopic:         true,
	BlobSidecarTopic:               true,
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
	for _, rawTopic := range req.Topics {
		splitTopic := strings.Split(rawTopic, ",")
		for _, topic := range splitTopic {
			if _, ok := casesHandled[topic]; !ok {
				return status.Errorf(codes.InvalidArgument, "Topic %s not allowed for event subscriptions", topic)
			}
			requestedTopics[topic] = true
		}
	}

	// Subscribe to event feeds from information received in the beacon node runtime.
	opsChan := make(chan *feed.Event, 1)
	opsSub := s.OperationNotifier.OperationFeed().Subscribe(opsChan)

	stateChan := make(chan *feed.Event, 1)
	stateSub := s.StateNotifier.StateFeed().Subscribe(stateChan)

	defer opsSub.Unsubscribe()
	defer stateSub.Unsubscribe()

	// Handle each event received and context cancelation.
	for {
		select {
		case event := <-opsChan:
			if err := handleBlockOperationEvents(stream, requestedTopics, event); err != nil {
				return status.Errorf(codes.Internal, "Could not handle block operations event: %v", err)
			}
		case event := <-stateChan:
			if err := s.handleStateEvents(stream, requestedTopics, event); err != nil {
				return status.Errorf(codes.Internal, "Could not handle state event: %v", err)
			}
		case <-s.Ctx.Done():
			return status.Errorf(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Errorf(codes.Canceled, "Context canceled")
		}
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
	case operation.BLSToExecutionChangeReceived:
		if _, ok := requestedTopics[BLSToExecutionChangeTopic]; !ok {
			return nil
		}
		changeData, ok := event.Data.(*operation.BLSToExecutionChangeReceivedData)
		if !ok {
			return nil
		}
		v2Change := migration.V1Alpha1SignedBLSToExecChangeToV2(changeData.Change)
		return streamData(stream, BLSToExecutionChangeTopic, v2Change)
	case operation.BlobSidecarReceived:
		if _, ok := requestedTopics[BlobSidecarTopic]; !ok {
			return nil
		}
		blobData, ok := event.Data.(*operation.BlobSidecarReceivedData)
		if !ok {
			return nil
		}
		if blobData == nil || blobData.Blob == nil {
			return nil
		}
		versionedHash := blockchain.ConvertKzgCommitmentToVersionedHash(blobData.Blob.Message.KzgCommitment)
		blobEvent := &ethpb.EventBlobSidecar{
			BlockRoot:     bytesutil.SafeCopyBytes(blobData.Blob.Message.BlockRoot),
			Index:         blobData.Blob.Message.Index,
			Slot:          blobData.Blob.Message.Slot,
			VersionedHash: bytesutil.SafeCopyBytes(versionedHash.Bytes()),
			KzgCommitment: bytesutil.SafeCopyBytes(blobData.Blob.Message.KzgCommitment),
		}
		return streamData(stream, BlobSidecarTopic, blobEvent)
	default:
		return nil
	}
}

func (s *Server) handleStateEvents(
	stream ethpbservice.Events_StreamEventsServer, requestedTopics map[string]bool, event *feed.Event,
) error {
	switch event.Type {
	case statefeed.NewHead:
		if _, ok := requestedTopics[HeadTopic]; ok {
			head, ok := event.Data.(*ethpb.EventHead)
			if !ok {
				return nil
			}
			return streamData(stream, HeadTopic, head)
		}
		if _, ok := requestedTopics[PayloadAttributesTopic]; ok {
			if err := s.streamPayloadAttributes(stream); err != nil {
				log.WithError(err).Error("Unable to obtain stream payload attributes")
			}
			return nil
		}
		return nil
	case statefeed.MissedSlot:
		if _, ok := requestedTopics[PayloadAttributesTopic]; ok {
			if err := s.streamPayloadAttributes(stream); err != nil {
				log.WithError(err).Error("Unable to obtain stream payload attributes")
			}
			return nil
		}
		return nil
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
	case statefeed.BlockProcessed:
		if _, ok := requestedTopics[BlockTopic]; !ok {
			return nil
		}
		blkData, ok := event.Data.(*statefeed.BlockProcessedData)
		if !ok {
			return nil
		}
		v1Data, err := migration.BlockIfaceToV1BlockHeader(blkData.SignedBlock)
		if err != nil {
			return err
		}
		item, err := v1Data.Message.HashTreeRoot()
		if err != nil {
			return errors.Wrap(err, "could not hash tree root block")
		}
		eventBlock := &ethpb.EventBlock{
			Slot:                blkData.Slot,
			Block:               item[:],
			ExecutionOptimistic: blkData.Optimistic,
		}
		return streamData(stream, BlockTopic, eventBlock)
	default:
		return nil
	}
}

// streamPayloadAttributes on new head event.
// This event stream is intended to be used by builders and relays.
// parent_ fields are based on state at N_{current_slot}, while the rest of fields are based on state of N_{current_slot + 1}
func (s *Server) streamPayloadAttributes(stream ethpbservice.Events_StreamEventsServer) error {
	headRoot, err := s.HeadFetcher.HeadRoot(s.Ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head root")
	}
	st, err := s.HeadFetcher.HeadState(s.Ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head state")
	}
	// advance the headstate
	headState, err := transition.ProcessSlotsIfPossible(s.Ctx, st, s.ChainInfoFetcher.CurrentSlot()+1)
	if err != nil {
		return err
	}

	headBlock, err := s.HeadFetcher.HeadBlock(s.Ctx)
	if err != nil {
		return err
	}

	headPayload, err := headBlock.Block().Body().Execution()
	if err != nil {
		return err
	}

	t, err := slots.ToTime(uint64(headState.GenesisTime()), headState.Slot())
	if err != nil {
		return err
	}

	prevRando, err := helpers.RandaoMix(headState, time.CurrentEpoch(headState))
	if err != nil {
		return err
	}

	proposerIndex, err := helpers.BeaconProposerIndex(s.Ctx, headState)
	if err != nil {
		return err
	}

	switch headState.Version() {
	case version.Bellatrix:
		return streamData(stream, PayloadAttributesTopic, &ethpb.EventPayloadAttributeV1{
			Version: version.String(headState.Version()),
			Data: &ethpb.EventPayloadAttributeV1_BasePayloadAttribute{
				ProposerIndex:     proposerIndex,
				ProposalSlot:      headState.Slot(),
				ParentBlockNumber: headPayload.BlockNumber(),
				ParentBlockRoot:   headRoot,
				ParentBlockHash:   headPayload.BlockHash(),
				PayloadAttributes: &enginev1.PayloadAttributes{
					Timestamp:             uint64(t.Unix()),
					PrevRandao:            prevRando,
					SuggestedFeeRecipient: headPayload.FeeRecipient(),
				},
			},
		})
	case version.Capella, version.Deneb:
		withdrawals, err := headState.ExpectedWithdrawals()
		if err != nil {
			return err
		}
		return streamData(stream, PayloadAttributesTopic, &ethpb.EventPayloadAttributeV2{
			Version: version.String(headState.Version()),
			Data: &ethpb.EventPayloadAttributeV2_BasePayloadAttribute{
				ProposerIndex:     proposerIndex,
				ProposalSlot:      headState.Slot(),
				ParentBlockNumber: headPayload.BlockNumber(),
				ParentBlockRoot:   headRoot,
				ParentBlockHash:   headPayload.BlockHash(),
				PayloadAttributes: &enginev1.PayloadAttributesV2{
					Timestamp:             uint64(t.Unix()),
					PrevRandao:            prevRando,
					SuggestedFeeRecipient: headPayload.FeeRecipient(),
					Withdrawals:           withdrawals,
				},
			},
		})
	default:
		return errors.New("payload version is not supported")
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
