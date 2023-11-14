package events

import (
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpbservice "github.com/prysmaticlabs/prysm/v4/proto/eth/service"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

const topicDataMismatch = "Event data type %T does not correspond to event topic %s"

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

// StreamEvents provides endpoint to subscribe to beacon node Server-Sent-Events stream.
// Consumers should use eventsource implementation to listen on those events.
// Servers may send SSE comments beginning with ':' for any purpose,
// including to keep the event stream connection alive in the presence of proxy servers.
func (s *Server) StreamEvents(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "events.StreamEvents")
	defer span.End()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http2.HandleError(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	topics := r.URL.Query()["topics"]
	if len(topics) == 0 {
		http2.HandleError(w, "No topics specified to subscribe to", http.StatusBadRequest)
		return
	}
	topicsMap := make(map[string]bool)
	for _, topic := range topics {
		if _, ok := casesHandled[topic]; !ok {
			http2.HandleError(w, fmt.Sprintf("Invalid topic: %s", topic), http.StatusBadRequest)
			return
		}
		topicsMap[topic] = true
	}

	// Subscribe to event feeds from information received in the beacon node runtime.
	opsChan := make(chan *feed.Event, 1)
	opsSub := s.OperationNotifier.OperationFeed().Subscribe(opsChan)
	stateChan := make(chan *feed.Event, 1)
	stateSub := s.StateNotifier.StateFeed().Subscribe(stateChan)
	defer opsSub.Unsubscribe()
	defer stateSub.Unsubscribe()

	// Set up SSE response headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Handle each event received and context cancellation.
	for {
		select {
		case event := <-opsChan:
			if err := handleBlockOperationEvents(w, flusher, topicsMap, event); err != nil {
				return status.Errorf(codes.Internal, "Could not handle block operations event: %v", err)
			}
		case event := <-stateChan:
			if err := s.handleStateEvents(w, flusher, topicsMap, event); err != nil {
				return status.Errorf(codes.Internal, "Could not handle state event: %v", err)
			}
		case <-s.Ctx.Done():
			return status.Errorf(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Errorf(codes.Canceled, "Context canceled")
		}
	}
}

func handleBlockOperationEvents(w http.ResponseWriter, flusher http.Flusher, requestedTopics map[string]bool, event *feed.Event) bool {
	switch event.Type {
	case operation.AggregatedAttReceived:
		if _, ok := requestedTopics[AttestationTopic]; !ok {
			return true
		}
		attData, ok := event.Data.(*operation.AggregatedAttReceivedData)
		if !ok {
			http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, AttestationTopic), http.StatusInternalServerError)
			return false
		}
		att := shared.AttestationFromConsensus(attData.Attestation.Aggregate)
		return send(w, flusher, AttestationTopic, att)
	case operation.UnaggregatedAttReceived:
		if _, ok := requestedTopics[AttestationTopic]; !ok {
			return true
		}
		attData, ok := event.Data.(*operation.UnAggregatedAttReceivedData)
		if !ok {
			http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, AttestationTopic), http.StatusInternalServerError)
			return false
		}
		att := shared.AttestationFromConsensus(attData.Attestation)
		return send(w, flusher, AttestationTopic, att)
	case operation.ExitReceived:
		if _, ok := requestedTopics[VoluntaryExitTopic]; !ok {
			return true
		}
		exitData, ok := event.Data.(*operation.ExitReceivedData)
		if !ok {
			http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, VoluntaryExitTopic), http.StatusInternalServerError)
			return false
		}
		exit := shared.SignedVoluntaryExitFromConsensus(exitData.Exit)
		return send(w, flusher, VoluntaryExitTopic, exit)
	case operation.SyncCommitteeContributionReceived:
		if _, ok := requestedTopics[SyncCommitteeContributionTopic]; !ok {
			return true
		}
		contributionData, ok := event.Data.(*operation.SyncCommitteeContributionReceivedData)
		if !ok {
			http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, SyncCommitteeContributionTopic), http.StatusInternalServerError)
			return false
		}
		contribution := shared.SignedContributionAndProofFromConsensus(contributionData.Contribution)
		return send(w, flusher, SyncCommitteeContributionTopic, contribution)
	case operation.BLSToExecutionChangeReceived:
		if _, ok := requestedTopics[BLSToExecutionChangeTopic]; !ok {
			return true
		}
		changeData, ok := event.Data.(*operation.BLSToExecutionChangeReceivedData)
		if !ok {
			http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, BLSToExecutionChangeTopic), http.StatusInternalServerError)
			return false
		}
		change, err := shared.SignedBlsToExecutionChangeFromConsensus(changeData.Change)
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return false
		}
		return send(w, flusher, BLSToExecutionChangeTopic, change)
	case operation.BlobSidecarReceived:
		if _, ok := requestedTopics[BlobSidecarTopic]; !ok {
			return true
		}
		blobData, ok := event.Data.(*operation.BlobSidecarReceivedData)
		if !ok {
			http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, BlobSidecarTopic), http.StatusInternalServerError)
			return false
		}
		versionedHash := blockchain.ConvertKzgCommitmentToVersionedHash(blobData.Blob.Message.KzgCommitment)
		blobEvent := &BlobSidecarEvent{
			BlockRoot:     hexutil.Encode(blobData.Blob.Message.BlockRoot),
			Index:         fmt.Sprintf("%d", blobData.Blob.Message.Index),
			Slot:          fmt.Sprintf("%d", blobData.Blob.Message.Slot),
			VersionedHash: versionedHash.String(),
			KzgCommitment: hexutil.Encode(blobData.Blob.Message.KzgCommitment),
		}
		return send(w, flusher, BlobSidecarTopic, blobEvent)
	default:
		return true
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
			return send(stream, HeadTopic, head)
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
		return send(stream, FinalizedCheckpointTopic, finalizedCheckpoint)
	case statefeed.Reorg:
		if _, ok := requestedTopics[ChainReorgTopic]; !ok {
			return nil
		}
		reorg, ok := event.Data.(*ethpb.EventChainReorg)
		if !ok {
			return nil
		}
		return send(stream, ChainReorgTopic, reorg)
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
		return send(stream, BlockTopic, eventBlock)
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

	t, err := slots.ToTime(headState.GenesisTime(), headState.Slot())
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
		return send(stream, PayloadAttributesTopic, &ethpb.EventPayloadAttributeV1{
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
	case version.Capella:
		withdrawals, err := headState.ExpectedWithdrawals()
		if err != nil {
			return err
		}
		return send(stream, PayloadAttributesTopic, &ethpb.EventPayloadAttributeV2{
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
	case version.Deneb:
		blargh
	default:
		return errors.New("payload version is not supported")
	}
}

func send(w http.ResponseWriter, flusher http.Flusher, name string, data interface{}) bool {
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	flusher.Flush()
	return true
}
