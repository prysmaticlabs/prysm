package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	time2 "time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
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
	// ProposerSlashingTopic represents a new proposer slashing event topic
	ProposerSlashingTopic = "proposer_slashing"
	// AttesterSlashingTopic represents a new attester slashing event topic
	AttesterSlashingTopic = "attester_slashing"
	// LightClientFinalityUpdateTopic represents a new light client finality update event topic.
	LightClientFinalityUpdateTopic = "light_client_finality_update"
	// LightClientOptimisticUpdateTopic represents a new light client optimistic update event topic.
	LightClientOptimisticUpdateTopic = "light_client_optimistic_update"
)

const topicDataMismatch = "Event data type %T does not correspond to event topic %s"

const chanBuffer = 1000

var casesHandled = map[string]bool{
	HeadTopic:                        true,
	BlockTopic:                       true,
	AttestationTopic:                 true,
	VoluntaryExitTopic:               true,
	FinalizedCheckpointTopic:         true,
	ChainReorgTopic:                  true,
	SyncCommitteeContributionTopic:   true,
	BLSToExecutionChangeTopic:        true,
	PayloadAttributesTopic:           true,
	BlobSidecarTopic:                 true,
	ProposerSlashingTopic:            true,
	AttesterSlashingTopic:            true,
	LightClientFinalityUpdateTopic:   true,
	LightClientOptimisticUpdateTopic: true,
}

// StreamEvents provides an endpoint to subscribe to the beacon node Server-Sent-Events stream.
// Consumers should use the eventsource implementation to listen for those events.
// Servers may send SSE comments beginning with ':' for any purpose,
// including to keep the event stream connection alive in the presence of proxy servers.
func (s *Server) StreamEvents(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "events.StreamEvents")
	defer span.End()

	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.HandleError(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	topics := r.URL.Query()["topics"]
	if len(topics) == 0 {
		httputil.HandleError(w, "No topics specified to subscribe to", http.StatusBadRequest)
		return
	}
	topicsMap := make(map[string]bool)
	for _, topic := range topics {
		if _, ok := casesHandled[topic]; !ok {
			httputil.HandleError(w, fmt.Sprintf("Invalid topic: %s", topic), http.StatusBadRequest)
			return
		}
		topicsMap[topic] = true
	}

	// Subscribe to event feeds from information received in the beacon node runtime.
	opsChan := make(chan *feed.Event, chanBuffer)
	opsSub := s.OperationNotifier.OperationFeed().Subscribe(opsChan)
	stateChan := make(chan *feed.Event, chanBuffer)
	stateSub := s.StateNotifier.StateFeed().Subscribe(stateChan)
	defer opsSub.Unsubscribe()
	defer stateSub.Unsubscribe()

	// Set up SSE response headers
	w.Header().Set("Content-Type", api.EventStreamMediaType)
	w.Header().Set("Connection", api.KeepAlive)

	// Handle each event received and context cancellation.
	// We send a keepalive dummy message immediately to prevent clients
	// stalling while waiting for the first response chunk.
	// After that we send a keepalive dummy message every SECONDS_PER_SLOT
	// to prevent anyone (e.g. proxy servers) from closing connections.
	if err := sendKeepalive(w, flusher); err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	keepaliveTicker := time2.NewTicker(time2.Duration(params.BeaconConfig().SecondsPerSlot) * time2.Second)

	for {
		select {
		case event := <-opsChan:
			if err := handleBlockOperationEvents(w, flusher, topicsMap, event); err != nil {
				httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case event := <-stateChan:
			if err := s.handleStateEvents(ctx, w, flusher, topicsMap, event); err != nil {
				httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case <-keepaliveTicker.C:
			if err := sendKeepalive(w, flusher); err != nil {
				httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func handleBlockOperationEvents(w http.ResponseWriter, flusher http.Flusher, requestedTopics map[string]bool, event *feed.Event) error {
	switch event.Type {
	case operation.AggregatedAttReceived:
		if _, ok := requestedTopics[AttestationTopic]; !ok {
			return nil
		}
		attData, ok := event.Data.(*operation.AggregatedAttReceivedData)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, AttestationTopic)
		}
		att := structs.AttFromConsensus(attData.Attestation.Aggregate)
		return send(w, flusher, AttestationTopic, att)
	case operation.UnaggregatedAttReceived:
		if _, ok := requestedTopics[AttestationTopic]; !ok {
			return nil
		}
		attData, ok := event.Data.(*operation.UnAggregatedAttReceivedData)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, AttestationTopic)
		}
		att := structs.AttFromConsensus(attData.Attestation)
		return send(w, flusher, AttestationTopic, att)
	case operation.ExitReceived:
		if _, ok := requestedTopics[VoluntaryExitTopic]; !ok {
			return nil
		}
		exitData, ok := event.Data.(*operation.ExitReceivedData)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, VoluntaryExitTopic)
		}
		exit := structs.SignedExitFromConsensus(exitData.Exit)
		return send(w, flusher, VoluntaryExitTopic, exit)
	case operation.SyncCommitteeContributionReceived:
		if _, ok := requestedTopics[SyncCommitteeContributionTopic]; !ok {
			return nil
		}
		contributionData, ok := event.Data.(*operation.SyncCommitteeContributionReceivedData)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, SyncCommitteeContributionTopic)
		}
		contribution := structs.SignedContributionAndProofFromConsensus(contributionData.Contribution)
		return send(w, flusher, SyncCommitteeContributionTopic, contribution)
	case operation.BLSToExecutionChangeReceived:
		if _, ok := requestedTopics[BLSToExecutionChangeTopic]; !ok {
			return nil
		}
		changeData, ok := event.Data.(*operation.BLSToExecutionChangeReceivedData)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, BLSToExecutionChangeTopic)
		}
		return send(w, flusher, BLSToExecutionChangeTopic, structs.SignedBLSChangeFromConsensus(changeData.Change))
	case operation.BlobSidecarReceived:
		if _, ok := requestedTopics[BlobSidecarTopic]; !ok {
			return nil
		}
		blobData, ok := event.Data.(*operation.BlobSidecarReceivedData)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, BlobSidecarTopic)
		}
		versionedHash := blockchain.ConvertKzgCommitmentToVersionedHash(blobData.Blob.KzgCommitment)
		blobEvent := &structs.BlobSidecarEvent{
			BlockRoot:     hexutil.Encode(blobData.Blob.BlockRootSlice()),
			Index:         fmt.Sprintf("%d", blobData.Blob.Index),
			Slot:          fmt.Sprintf("%d", blobData.Blob.Slot()),
			VersionedHash: versionedHash.String(),
			KzgCommitment: hexutil.Encode(blobData.Blob.KzgCommitment),
		}
		return send(w, flusher, BlobSidecarTopic, blobEvent)
	case operation.AttesterSlashingReceived:
		if _, ok := requestedTopics[AttesterSlashingTopic]; !ok {
			return nil
		}
		attesterSlashingData, ok := event.Data.(*operation.AttesterSlashingReceivedData)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, AttesterSlashingTopic)
		}
		return send(w, flusher, AttesterSlashingTopic, structs.AttesterSlashingFromConsensus(attesterSlashingData.AttesterSlashing))
	case operation.ProposerSlashingReceived:
		if _, ok := requestedTopics[ProposerSlashingTopic]; !ok {
			return nil
		}
		proposerSlashingData, ok := event.Data.(*operation.ProposerSlashingReceivedData)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, ProposerSlashingTopic)
		}
		return send(w, flusher, ProposerSlashingTopic, structs.ProposerSlashingFromConsensus(proposerSlashingData.ProposerSlashing))
	}
	return nil
}

func (s *Server) handleStateEvents(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, requestedTopics map[string]bool, event *feed.Event) error {
	switch event.Type {
	case statefeed.NewHead:
		if _, ok := requestedTopics[HeadTopic]; ok {
			headData, ok := event.Data.(*ethpb.EventHead)
			if !ok {
				return write(w, flusher, topicDataMismatch, event.Data, HeadTopic)
			}
			head := &structs.HeadEvent{
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
			return s.sendPayloadAttributes(ctx, w, flusher)
		}
	case statefeed.MissedSlot:
		if _, ok := requestedTopics[PayloadAttributesTopic]; ok {
			return s.sendPayloadAttributes(ctx, w, flusher)
		}
	case statefeed.FinalizedCheckpoint:
		if _, ok := requestedTopics[FinalizedCheckpointTopic]; !ok {
			return nil
		}
		checkpointData, ok := event.Data.(*ethpb.EventFinalizedCheckpoint)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, FinalizedCheckpointTopic)
		}
		checkpoint := &structs.FinalizedCheckpointEvent{
			Block:               hexutil.Encode(checkpointData.Block),
			State:               hexutil.Encode(checkpointData.State),
			Epoch:               fmt.Sprintf("%d", checkpointData.Epoch),
			ExecutionOptimistic: checkpointData.ExecutionOptimistic,
		}
		return send(w, flusher, FinalizedCheckpointTopic, checkpoint)
	case statefeed.LightClientFinalityUpdate:
		if _, ok := requestedTopics[LightClientFinalityUpdateTopic]; !ok {
			return nil
		}
		updateData, ok := event.Data.(*ethpbv2.LightClientFinalityUpdateWithVersion)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, LightClientFinalityUpdateTopic)
		}

		var finalityBranch []string
		for _, b := range updateData.Data.FinalityBranch {
			finalityBranch = append(finalityBranch, hexutil.Encode(b))
		}
		update := &structs.LightClientFinalityUpdateEvent{
			Version: version.String(int(updateData.Version)),
			Data: &structs.LightClientFinalityUpdate{
				AttestedHeader: &structs.BeaconBlockHeader{
					Slot:          fmt.Sprintf("%d", updateData.Data.AttestedHeader.Slot),
					ProposerIndex: fmt.Sprintf("%d", updateData.Data.AttestedHeader.ProposerIndex),
					ParentRoot:    hexutil.Encode(updateData.Data.AttestedHeader.ParentRoot),
					StateRoot:     hexutil.Encode(updateData.Data.AttestedHeader.StateRoot),
					BodyRoot:      hexutil.Encode(updateData.Data.AttestedHeader.BodyRoot),
				},
				FinalizedHeader: &structs.BeaconBlockHeader{
					Slot:          fmt.Sprintf("%d", updateData.Data.FinalizedHeader.Slot),
					ProposerIndex: fmt.Sprintf("%d", updateData.Data.FinalizedHeader.ProposerIndex),
					ParentRoot:    hexutil.Encode(updateData.Data.FinalizedHeader.ParentRoot),
					StateRoot:     hexutil.Encode(updateData.Data.FinalizedHeader.StateRoot),
				},
				FinalityBranch: finalityBranch,
				SyncAggregate: &structs.SyncAggregate{
					SyncCommitteeBits:      hexutil.Encode(updateData.Data.SyncAggregate.SyncCommitteeBits),
					SyncCommitteeSignature: hexutil.Encode(updateData.Data.SyncAggregate.SyncCommitteeSignature),
				},
				SignatureSlot: fmt.Sprintf("%d", updateData.Data.SignatureSlot),
			},
		}
		return send(w, flusher, LightClientFinalityUpdateTopic, update)
	case statefeed.LightClientOptimisticUpdate:
		if _, ok := requestedTopics[LightClientOptimisticUpdateTopic]; !ok {
			return nil
		}
		updateData, ok := event.Data.(*ethpbv2.LightClientOptimisticUpdateWithVersion)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, LightClientOptimisticUpdateTopic)
		}
		update := &structs.LightClientOptimisticUpdateEvent{
			Version: version.String(int(updateData.Version)),
			Data: &structs.LightClientOptimisticUpdate{
				AttestedHeader: &structs.BeaconBlockHeader{
					Slot:          fmt.Sprintf("%d", updateData.Data.AttestedHeader.Slot),
					ProposerIndex: fmt.Sprintf("%d", updateData.Data.AttestedHeader.ProposerIndex),
					ParentRoot:    hexutil.Encode(updateData.Data.AttestedHeader.ParentRoot),
					StateRoot:     hexutil.Encode(updateData.Data.AttestedHeader.StateRoot),
					BodyRoot:      hexutil.Encode(updateData.Data.AttestedHeader.BodyRoot),
				},
				SyncAggregate: &structs.SyncAggregate{
					SyncCommitteeBits:      hexutil.Encode(updateData.Data.SyncAggregate.SyncCommitteeBits),
					SyncCommitteeSignature: hexutil.Encode(updateData.Data.SyncAggregate.SyncCommitteeSignature),
				},
				SignatureSlot: fmt.Sprintf("%d", updateData.Data.SignatureSlot),
			},
		}
		return send(w, flusher, LightClientOptimisticUpdateTopic, update)
	case statefeed.Reorg:
		if _, ok := requestedTopics[ChainReorgTopic]; !ok {
			return nil
		}
		reorgData, ok := event.Data.(*ethpb.EventChainReorg)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, ChainReorgTopic)
		}
		reorg := &structs.ChainReorgEvent{
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
			return nil
		}
		blkData, ok := event.Data.(*statefeed.BlockProcessedData)
		if !ok {
			return write(w, flusher, topicDataMismatch, event.Data, BlockTopic)
		}
		blockRoot, err := blkData.SignedBlock.Block().HashTreeRoot()
		if err != nil {
			return write(w, flusher, "Could not get block root: "+err.Error())
		}
		blk := &structs.BlockEvent{
			Slot:                fmt.Sprintf("%d", blkData.Slot),
			Block:               hexutil.Encode(blockRoot[:]),
			ExecutionOptimistic: blkData.Optimistic,
		}
		return send(w, flusher, BlockTopic, blk)
	}
	return nil
}

// This event stream is intended to be used by builders and relays.
// Parent fields are based on state at N_{current_slot}, while the rest of fields are based on state of N_{current_slot + 1}
func (s *Server) sendPayloadAttributes(ctx context.Context, w http.ResponseWriter, flusher http.Flusher) error {
	headRoot, err := s.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return write(w, flusher, "Could not get head root: "+err.Error())
	}
	st, err := s.HeadFetcher.HeadState(ctx)
	if err != nil {
		return write(w, flusher, "Could not get head state: "+err.Error())
	}
	// advance the head state
	headState, err := transition.ProcessSlotsIfPossible(ctx, st, s.ChainInfoFetcher.CurrentSlot()+1)
	if err != nil {
		return write(w, flusher, "Could not advance head state: "+err.Error())
	}

	headBlock, err := s.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return write(w, flusher, "Could not get head block: "+err.Error())
	}

	headPayload, err := headBlock.Block().Body().Execution()
	if err != nil {
		return write(w, flusher, "Could not get execution payload: "+err.Error())
	}

	t, err := slots.ToTime(headState.GenesisTime(), headState.Slot())
	if err != nil {
		return write(w, flusher, "Could not get head state slot time: "+err.Error())
	}

	prevRando, err := helpers.RandaoMix(headState, time.CurrentEpoch(headState))
	if err != nil {
		return write(w, flusher, "Could not get head state randao mix: "+err.Error())
	}

	proposerIndex, err := helpers.BeaconProposerIndex(ctx, headState)
	if err != nil {
		return write(w, flusher, "Could not get head state proposer index: "+err.Error())
	}

	var attributes interface{}
	switch headState.Version() {
	case version.Bellatrix:
		attributes = &structs.PayloadAttributesV1{
			Timestamp:             fmt.Sprintf("%d", t.Unix()),
			PrevRandao:            hexutil.Encode(prevRando),
			SuggestedFeeRecipient: hexutil.Encode(headPayload.FeeRecipient()),
		}
	case version.Capella:
		withdrawals, err := headState.ExpectedWithdrawals()
		if err != nil {
			return write(w, flusher, "Could not get head state expected withdrawals: "+err.Error())
		}
		attributes = &structs.PayloadAttributesV2{
			Timestamp:             fmt.Sprintf("%d", t.Unix()),
			PrevRandao:            hexutil.Encode(prevRando),
			SuggestedFeeRecipient: hexutil.Encode(headPayload.FeeRecipient()),
			Withdrawals:           structs.WithdrawalsFromConsensus(withdrawals),
		}
	case version.Deneb:
		withdrawals, err := headState.ExpectedWithdrawals()
		if err != nil {
			return write(w, flusher, "Could not get head state expected withdrawals: "+err.Error())
		}
		parentRoot, err := headBlock.Block().HashTreeRoot()
		if err != nil {
			return write(w, flusher, "Could not get head block root: "+err.Error())
		}
		attributes = &structs.PayloadAttributesV3{
			Timestamp:             fmt.Sprintf("%d", t.Unix()),
			PrevRandao:            hexutil.Encode(prevRando),
			SuggestedFeeRecipient: hexutil.Encode(headPayload.FeeRecipient()),
			Withdrawals:           structs.WithdrawalsFromConsensus(withdrawals),
			ParentBeaconBlockRoot: hexutil.Encode(parentRoot[:]),
		}
	default:
		return write(w, flusher, "Payload version %s is not supported", version.String(headState.Version()))
	}

	attributesBytes, err := json.Marshal(attributes)
	if err != nil {
		return write(w, flusher, err.Error())
	}
	eventData := structs.PayloadAttributesEventData{
		ProposerIndex:     fmt.Sprintf("%d", proposerIndex),
		ProposalSlot:      fmt.Sprintf("%d", headState.Slot()),
		ParentBlockNumber: fmt.Sprintf("%d", headPayload.BlockNumber()),
		ParentBlockRoot:   hexutil.Encode(headRoot),
		ParentBlockHash:   hexutil.Encode(headPayload.BlockHash()),
		PayloadAttributes: attributesBytes,
	}
	eventDataBytes, err := json.Marshal(eventData)
	if err != nil {
		return write(w, flusher, err.Error())
	}
	return send(w, flusher, PayloadAttributesTopic, &structs.PayloadAttributesEvent{
		Version: version.String(headState.Version()),
		Data:    eventDataBytes,
	})
}

func send(w http.ResponseWriter, flusher http.Flusher, name string, data interface{}) error {
	j, err := json.Marshal(data)
	if err != nil {
		return write(w, flusher, "Could not marshal event to JSON: "+err.Error())
	}
	return write(w, flusher, "event: %s\ndata: %s\n\n", name, string(j))
}

func sendKeepalive(w http.ResponseWriter, flusher http.Flusher) error {
	return write(w, flusher, ":\n\n")
}

func write(w http.ResponseWriter, flusher http.Flusher, format string, a ...any) error {
	_, err := fmt.Fprintf(w, format, a...)
	if err != nil {
		return errors.Wrap(err, "could not write to response writer")
	}
	flusher.Flush()
	return nil
}
