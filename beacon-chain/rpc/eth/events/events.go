package events

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
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
	ctx, span := trace.StartSpan(r.Context(), "events.StreamEvents")
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
			if ok = handleBlockOperationEvents(w, flusher, topicsMap, event); !ok {
				return
			}
		case event := <-stateChan:
			if ok = s.handleStateEvents(w, flusher, topicsMap, event); !ok {
				return
			}
		case <-s.Ctx.Done():
			return
		case <-ctx.Done():
			return
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

func (s *Server) handleStateEvents(w http.ResponseWriter, flusher http.Flusher, requestedTopics map[string]bool, event *feed.Event) bool {
	switch event.Type {
	case statefeed.NewHead:
		if _, ok := requestedTopics[HeadTopic]; ok {
			headData, ok := event.Data.(*ethpb.EventHead)
			if !ok {
				http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, HeadTopic), http.StatusInternalServerError)
				return false
			}
			head := &HeadEvent{
				Slot:                      fmt.Sprintf("%d", headData.Slot),
				Block:                     hexutil.Encode(headData.Block),
				State:                     hexutil.Encode(headData.State),
				EpochTransition:           headData.EpochTransition,
				ExecutionOptimistic:       headData.ExecutionOptimistic,
				PreviousDutyDependentRoot: hexutil.Encode(headData.PreviousDutyDependentRoot),
				CurrentDutyDependentRoot:  hexutil.Encode(headData.CurrentDutyDependentRoot),
			}
			return send(w, flusher, HeadTopic, head)
		}
		if _, ok := requestedTopics[PayloadAttributesTopic]; ok {
			if ok = s.sendPayloadAttributes(w, flusher); !ok {
				return false
			}
		}
		return true
	case statefeed.MissedSlot:
		if _, ok := requestedTopics[PayloadAttributesTopic]; ok {
			if ok = s.sendPayloadAttributes(w, flusher); !ok {
				return false
			}
		}
		return true
	case statefeed.FinalizedCheckpoint:
		if _, ok := requestedTopics[FinalizedCheckpointTopic]; !ok {
			return true
		}
		checkpointData, ok := event.Data.(*ethpb.EventFinalizedCheckpoint)
		if !ok {
			http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, FinalizedCheckpointTopic), http.StatusInternalServerError)
			return false
		}
		checkpoint := &FinalizedCheckpointEvent{
			Block:               hexutil.Encode(checkpointData.Block),
			State:               hexutil.Encode(checkpointData.State),
			Epoch:               fmt.Sprintf("%d", checkpointData.Epoch),
			ExecutionOptimistic: checkpointData.ExecutionOptimistic,
		}
		return send(w, flusher, FinalizedCheckpointTopic, checkpoint)
	case statefeed.Reorg:
		if _, ok := requestedTopics[ChainReorgTopic]; !ok {
			return true
		}
		reorgData, ok := event.Data.(*ethpb.EventChainReorg)
		if !ok {
			http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, ChainReorgTopic), http.StatusInternalServerError)
			return false
		}
		reorg := &ChainReorgEvent{
			Slot:                fmt.Sprintf("%d", reorgData.Slot),
			Depth:               fmt.Sprintf("%d", reorgData.Depth),
			OldHeadBlock:        hexutil.Encode(reorgData.OldHeadBlock),
			NewHeadBlock:        hexutil.Encode(reorgData.NewHeadBlock),
			OldHeadState:        hexutil.Encode(reorgData.OldHeadState),
			NewHeadState:        hexutil.Encode(reorgData.NewHeadState),
			Epoch:               fmt.Sprintf("%d", reorgData.Epoch),
			ExecutionOptimistic: reorgData.ExecutionOptimistic,
		}
		return send(w, flusher, ChainReorgTopic, reorg)
	case statefeed.BlockProcessed:
		if _, ok := requestedTopics[BlockTopic]; !ok {
			return true
		}
		blkData, ok := event.Data.(*statefeed.BlockProcessedData)
		if !ok {
			http2.HandleError(w, fmt.Sprintf(topicDataMismatch, event.Data, BlockTopic), http.StatusInternalServerError)
			return false
		}
		blockRoot, err := blkData.SignedBlock.Block().HashTreeRoot()
		if err != nil {
			http2.HandleError(w, "Could not get block root: "+err.Error(), http.StatusInternalServerError)
			return false
		}
		blk := &BlockEvent{
			Slot:                fmt.Sprintf("%d", blkData.Slot),
			Block:               hexutil.Encode(blockRoot[:]),
			ExecutionOptimistic: blkData.Optimistic,
		}
		return send(w, flusher, BlockTopic, blk)
	default:
		return true
	}
}

// This event stream is intended to be used by builders and relays.
// Parent fields are based on state at N_{current_slot}, while the rest of fields are based on state of N_{current_slot + 1}
func (s *Server) sendPayloadAttributes(w http.ResponseWriter, flusher http.Flusher) bool {
	headRoot, err := s.HeadFetcher.HeadRoot(s.Ctx)
	if err != nil {
		http2.HandleError(w, "Could not get head root: "+err.Error(), http.StatusInternalServerError)
		return false
	}
	st, err := s.HeadFetcher.HeadState(s.Ctx)
	if err != nil {
		http2.HandleError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return false
	}
	// advance the head state
	headState, err := transition.ProcessSlotsIfPossible(s.Ctx, st, s.ChainInfoFetcher.CurrentSlot()+1)
	if err != nil {
		http2.HandleError(w, "Could not advance head state: "+err.Error(), http.StatusInternalServerError)
		return false
	}

	headBlock, err := s.HeadFetcher.HeadBlock(s.Ctx)
	if err != nil {
		http2.HandleError(w, "Could not get head block: "+err.Error(), http.StatusInternalServerError)
		return false
	}

	headPayload, err := headBlock.Block().Body().Execution()
	if err != nil {
		http2.HandleError(w, "Could not get execution payload: "+err.Error(), http.StatusInternalServerError)
		return false
	}

	t, err := slots.ToTime(headState.GenesisTime(), headState.Slot())
	if err != nil {
		http2.HandleError(w, "Could not get head state slot time: "+err.Error(), http.StatusInternalServerError)
		return false
	}

	prevRando, err := helpers.RandaoMix(headState, time.CurrentEpoch(headState))
	if err != nil {
		http2.HandleError(w, "Could not get head state randao mix: "+err.Error(), http.StatusInternalServerError)
		return false
	}

	proposerIndex, err := helpers.BeaconProposerIndex(s.Ctx, headState)
	if err != nil {
		http2.HandleError(w, "Could not get head state proposer index: "+err.Error(), http.StatusInternalServerError)
		return false
	}

	var attributes interface{}
	switch headState.Version() {
	case version.Bellatrix:
		attributes = &PayloadAttributesV1{
			Timestamp:             fmt.Sprintf("%d", t.Unix()),
			PrevRandao:            hexutil.Encode(prevRando),
			SuggestedFeeRecipient: hexutil.Encode(headPayload.FeeRecipient()),
		}
	case version.Capella:
		withdrawals, err := headState.ExpectedWithdrawals()
		if err != nil {
			http2.HandleError(w, "Could not get head state expected withdrawals: "+err.Error(), http.StatusInternalServerError)
			return false
		}
		attributes = &PayloadAttributesV2{
			Timestamp:             fmt.Sprintf("%d", t.Unix()),
			PrevRandao:            hexutil.Encode(prevRando),
			SuggestedFeeRecipient: hexutil.Encode(headPayload.FeeRecipient()),
			Withdrawals:           shared.WithdrawalsFromConsensus(withdrawals),
		}
	case version.Deneb:
		withdrawals, err := headState.ExpectedWithdrawals()
		if err != nil {
			http2.HandleError(w, "Could not get head state expected withdrawals: "+err.Error(), http.StatusInternalServerError)
			return false
		}
		parentRoot, err := headBlock.Block().HashTreeRoot()
		if err != nil {
			http2.HandleError(w, "Could not get head block root: "+err.Error(), http.StatusInternalServerError)
			return false
		}
		attributes = &PayloadAttributesV3{
			Timestamp:             fmt.Sprintf("%d", t.Unix()),
			PrevRandao:            hexutil.Encode(prevRando),
			SuggestedFeeRecipient: hexutil.Encode(headPayload.FeeRecipient()),
			Withdrawals:           shared.WithdrawalsFromConsensus(withdrawals),
			ParentBeaconBlockRoot: hexutil.Encode(parentRoot[:]),
		}
	default:
		http2.HandleError(w, fmt.Sprintf("Payload version %s is not supported", version.String(headState.Version())), http.StatusInternalServerError)
		return false
	}

	attributesBytes, err := json.Marshal(attributes)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	eventData := PayloadAttributesEventData{
		ProposerIndex:     fmt.Sprintf("%d", proposerIndex),
		ProposalSlot:      fmt.Sprintf("%d", headState.Slot()),
		ParentBlockNumber: fmt.Sprintf("%d", headPayload.BlockNumber()),
		ParentBlockRoot:   hexutil.Encode(headRoot),
		ParentBlockHash:   hexutil.Encode(headPayload.BlockHash()),
		PayloadAttributes: attributesBytes,
	}
	eventDataBytes, err := json.Marshal(eventData)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return send(w, flusher, PayloadAttributesTopic, &PayloadAttributesEvent{
		Version: version.String(headState.Version()),
		Data:    eventDataBytes,
	})
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
