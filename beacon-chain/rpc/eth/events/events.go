package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	chaintime "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	log "github.com/sirupsen/logrus"
)

const eventFeedDepth = 1000

const (
	InvalidTopic = "__invalid__"
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

var (
	errInvalidTopicName   = errors.New("invalid topic name")
	errNoValidTopics      = errors.New("no valid topics specified")
	errSlowReader         = errors.New("client failed to read fast enough to keep outgoing buffer below threshold")
	errNotRequested       = errors.New("event not requested by client")
	errUnhandledEventData = errors.New("unable to represent event data in the event stream")
)

// StreamingResponseWriter defines a type that can be used by the eventStreamer.
// This must be an http.ResponseWriter that supports flushing and hijacking.
type StreamingResponseWriter interface {
	http.ResponseWriter
	http.Flusher
}

// The eventStreamer uses lazyReaders to defer serialization until the moment the value is ready to be written to the client.
type lazyReader func() io.Reader

var opsFeedEventTopics = map[feed.EventType]string{
	operation.AggregatedAttReceived:             AttestationTopic,
	operation.UnaggregatedAttReceived:           AttestationTopic,
	operation.ExitReceived:                      VoluntaryExitTopic,
	operation.SyncCommitteeContributionReceived: SyncCommitteeContributionTopic,
	operation.BLSToExecutionChangeReceived:      BLSToExecutionChangeTopic,
	operation.BlobSidecarReceived:               BlobSidecarTopic,
	operation.AttesterSlashingReceived:          AttesterSlashingTopic,
	operation.ProposerSlashingReceived:          ProposerSlashingTopic,
}

var stateFeedEventTopics = map[feed.EventType]string{
	statefeed.NewHead:                     HeadTopic,
	statefeed.MissedSlot:                  PayloadAttributesTopic,
	statefeed.FinalizedCheckpoint:         FinalizedCheckpointTopic,
	statefeed.LightClientFinalityUpdate:   LightClientFinalityUpdateTopic,
	statefeed.LightClientOptimisticUpdate: LightClientOptimisticUpdateTopic,
	statefeed.Reorg:                       ChainReorgTopic,
	statefeed.BlockProcessed:              BlockTopic,
}

var topicsForStateFeed = topicsForFeed(stateFeedEventTopics)
var topicsForOpsFeed = topicsForFeed(opsFeedEventTopics)

func topicsForFeed(em map[feed.EventType]string) map[string]bool {
	topics := make(map[string]bool, len(em))
	for _, topic := range em {
		topics[topic] = true
	}
	return topics
}

type topicRequest struct {
	topics        map[string]bool
	needStateFeed bool
	needOpsFeed   bool
}

func (req *topicRequest) requested(topic string) bool {
	return req.topics[topic]
}

func newTopicRequest(topics []string) (*topicRequest, error) {
	req := &topicRequest{topics: make(map[string]bool)}
	for _, name := range topics {
		if topicsForStateFeed[name] {
			req.needStateFeed = true
		} else if topicsForOpsFeed[name] {
			req.needOpsFeed = true
		} else {
			return nil, errors.Wrapf(errInvalidTopicName, name)
		}
		req.topics[name] = true
	}
	if len(req.topics) == 0 || (!req.needStateFeed && !req.needOpsFeed) {
		return nil, errNoValidTopics
	}

	return req, nil
}

// StreamEvents provides an endpoint to subscribe to the beacon node Server-Sent-Events stream.
// Consumers should use the eventsource implementation to listen for those events.
// Servers may send SSE comments beginning with ':' for any purpose,
// including to keep the event stream connection alive in the presence of proxy servers.
func (s *Server) StreamEvents(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "events.StreamEvents")
	defer span.End()

	topics, err := newTopicRequest(r.URL.Query()["topics"])
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusBadRequest)
		return
	}

	sw, ok := w.(StreamingResponseWriter)
	if !ok {
		msg := "beacon node misconfiguration: http stack may not support required response handling features, like flushing"
		httputil.HandleError(w, msg, http.StatusInternalServerError)
		return
	}
	es, err := newEventStreamer(eventFeedDepth, s.KeepAliveInterval)
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := es.streamEvents(ctx, sw, topics, s); err != nil {
		log.WithError(err).Debug("Event streamer shutting down due to error.")
	}
}

func newEventStreamer(buffSize int, ka time.Duration) (*eventStreamer, error) {
	if ka == 0 {
		ka = time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second
	}
	return &eventStreamer{
		outbox:    make(chan lazyReader, buffSize),
		keepAlive: ka,
	}, nil
}

type eventStreamer struct {
	outbox    chan lazyReader
	keepAlive time.Duration
}

func (es *eventStreamer) streamEvents(ctx context.Context, w StreamingResponseWriter, req *topicRequest, s *Server) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go es.recvEventLoop(ctx, cancel, req, s)
	api.SetSSEHeaders(w)
	return es.outboxWriteLoop(ctx, w)
}

func (es *eventStreamer) recvEventLoop(ctx context.Context, cancel context.CancelFunc, req *topicRequest, s *Server) {
	eventsChan := make(chan *feed.Event, len(es.outbox))
	if req.needOpsFeed {
		opsSub := s.OperationNotifier.OperationFeed().Subscribe(eventsChan)
		defer opsSub.Unsubscribe()
	}
	if req.needStateFeed {
		stateSub := s.StateNotifier.StateFeed().Subscribe(eventsChan)
		defer stateSub.Unsubscribe()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-eventsChan:
			lr, err := s.lazyReaderForEvent(ctx, event, req)
			if err != nil {
				if !errors.Is(err, errNotRequested) {
					log.WithError(err).Error("StreamEvents API endpoint received an event it was unable to handle.")
				}
				continue
			}
			// If the client can't keep up, the outbox will eventually completely fill, at which
			// safeWrite will error, and we'll hit the below return statement, at which point the deferred
			// Unsuscribe calls will be made and the event feed will stop writing to this channel.
			// Since the outbox and event stream channels are separately buffered, the event subscription
			// channel should stay relatively empty, which gives this loop time to unsubscribe
			// and cleanup before the event stream channel fills and disrupts other readers.
			if err := es.safeWrite(ctx, lr); err != nil {
				cancel()
				// note: we could hijack the connection and close it here. Does that cause issues? What are the benefits?
				// A benefit of hijack and close is that it may force an error on the remote end, however just closing the context of the
				// http handler may be sufficient to cause the remote http response reader to close.
				log.WithField("event_type", fmt.Sprintf("%v", event.Data)).Warn("Unable to safely write event to stream, shutting down.")
				return
			}
		}
	}
}

func (es *eventStreamer) safeWrite(ctx context.Context, rf func() io.Reader) error {
	if rf == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case es.outbox <- rf:
		return nil
	default:
		// If this is the case, the select case to write to the outbox could not proceed, meaning the outbox is full.
		// If a reader can't keep up with the stream, we shut them down.
		return errSlowReader
	}
}

// newlineReader is used to write keep-alives to the client.
// keep-alives in the sse protocol are a single ':' colon followed by 2 newlines.
func newlineReader() io.Reader {
	return bytes.NewBufferString(":\n\n")
}

// outboxWriteLoop runs in a separate goroutine. Its job is to write the values in the outbox to
// the client as fast as the client can read them.
func (es *eventStreamer) outboxWriteLoop(ctx context.Context, w StreamingResponseWriter) error {
	// Write a keepalive at the start to test the connection and simplify test setup.
	if err := es.writeOutbox(ctx, w, nil); err != nil {
		return err
	}

	kaT := time.NewTimer(es.keepAlive)
	// Ensure the keepalive timer is stopped and drained if it has fired.
	defer func() {
		if !kaT.Stop() {
			<-kaT.C
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-kaT.C:
			if err := es.writeOutbox(ctx, w, nil); err != nil {
				return err
			}
			// In this case the timer doesn't need to be Stopped before the Reset call after the select statement,
			// because the timer has already fired.
		case lr := <-es.outbox:
			if err := es.writeOutbox(ctx, w, lr); err != nil {
				return err
			}
			// We don't know if the timer fired concurrently to this case being ready, so we need to check the return
			// of Stop and drain the timer channel if it fired. We won't need to do this in go 1.23.
			if !kaT.Stop() {
				<-kaT.C
			}
		}
		kaT.Reset(es.keepAlive)
	}
}

func (es *eventStreamer) writeOutbox(ctx context.Context, w StreamingResponseWriter, first lazyReader) error {
	needKeepAlive := true
	if first != nil {
		if _, err := io.Copy(w, first()); err != nil {
			return err
		}
		needKeepAlive = false
	}
	// While the first event was being read by the client, further events may be queued in the outbox.
	// We can drain them right away rather than go back out to the outer select statement, where the keepAlive timer
	// may have fired, triggering an unnecessary extra keep-alive write and flush.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case rf := <-es.outbox:
			if _, err := io.Copy(w, rf()); err != nil {
				return err
			}
			needKeepAlive = false
		default:
			if needKeepAlive {
				if _, err := io.Copy(w, newlineReader()); err != nil {
					return err
				}
			}
			w.Flush()
			return nil
		}
	}
}

func jsonMarshalReader(name string, v any) io.Reader {
	d, err := json.Marshal(v)
	if err != nil {
		log.WithError(err).WithField("type_name", fmt.Sprintf("%T", v)).Error("Could not marshal event data.")
		return nil
	}
	return bytes.NewBufferString("event: " + name + "\ndata: " + string(d) + "\n\n")
}

func topicForEvent(event *feed.Event) string {
	switch event.Data.(type) {
	case *operation.AggregatedAttReceivedData:
		return AttestationTopic
	case *operation.UnAggregatedAttReceivedData:
		return AttestationTopic
	case *operation.ExitReceivedData:
		return VoluntaryExitTopic
	case *operation.SyncCommitteeContributionReceivedData:
		return SyncCommitteeContributionTopic
	case *operation.BLSToExecutionChangeReceivedData:
		return BLSToExecutionChangeTopic
	case *operation.BlobSidecarReceivedData:
		return BlobSidecarTopic
	case *operation.AttesterSlashingReceivedData:
		return AttesterSlashingTopic
	case *operation.ProposerSlashingReceivedData:
		return ProposerSlashingTopic
	case *ethpb.EventHead:
		return HeadTopic
	case *ethpb.EventFinalizedCheckpoint:
		return FinalizedCheckpointTopic
	case *ethpbv2.LightClientFinalityUpdateWithVersion:
		return LightClientFinalityUpdateTopic
	case *ethpbv2.LightClientOptimisticUpdateWithVersion:
		return LightClientOptimisticUpdateTopic
	case *ethpb.EventChainReorg:
		return ChainReorgTopic
	case *statefeed.BlockProcessedData:
		return BlockTopic
	default:
		if event.Type == statefeed.MissedSlot {
			return PayloadAttributesTopic
		}
		return InvalidTopic
	}
}

func (s *Server) lazyReaderForEvent(ctx context.Context, event *feed.Event, topics *topicRequest) (lazyReader, error) {
	eventName := topicForEvent(event)
	if !topics.requested(eventName) {
		return nil, errNotRequested
	}
	if eventName == PayloadAttributesTopic {
		return s.currentPayloadAttributes(ctx)
	}
	switch v := event.Data.(type) {
	case *ethpb.EventHead:
		// The head event is a special case because, if the client requested the payload attributes topic,
		// we send two event messages in reaction; the head event and the payload attributes.
		headReader := func() io.Reader {
			return jsonMarshalReader(eventName, structs.HeadEventFromV1(v))
		}
		// Don't do the expensive attr lookup unless the client requested it.
		if !topics.requested(PayloadAttributesTopic) {
			return headReader, nil
		}
		// Since payload attributes could change before the outbox is written, we need to do a blocking operation to
		// get the current payload attributes right here.
		attrReader, err := s.currentPayloadAttributes(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get payload attributes for head event")
		}
		return func() io.Reader {
			return io.MultiReader(headReader(), attrReader())
		}, nil
	case *operation.AggregatedAttReceivedData:
		return func() io.Reader {
			att := structs.AttFromConsensus(v.Attestation.Aggregate)
			return jsonMarshalReader(eventName, att)
		}, nil
	case *operation.UnAggregatedAttReceivedData:
		att, ok := v.Attestation.(*eth.Attestation)
		if !ok {
			return nil, errors.Wrapf(errUnhandledEventData, "Unexpected type %T for the .Attestation field of UnAggregatedAttReceivedData", v.Attestation)
		}
		return func() io.Reader {
			att := structs.AttFromConsensus(att)
			return jsonMarshalReader(eventName, att)
		}, nil
	case *operation.ExitReceivedData:
		return func() io.Reader {
			return jsonMarshalReader(eventName, structs.SignedExitFromConsensus(v.Exit))
		}, nil
	case *operation.SyncCommitteeContributionReceivedData:
		return func() io.Reader {
			return jsonMarshalReader(eventName, structs.SignedContributionAndProofFromConsensus(v.Contribution))
		}, nil
	case *operation.BLSToExecutionChangeReceivedData:
		return func() io.Reader {
			return jsonMarshalReader(eventName, structs.SignedBLSChangeFromConsensus(v.Change))
		}, nil
	case *operation.BlobSidecarReceivedData:
		return func() io.Reader {
			versionedHash := blockchain.ConvertKzgCommitmentToVersionedHash(v.Blob.KzgCommitment)
			return jsonMarshalReader(eventName, &structs.BlobSidecarEvent{
				BlockRoot:     hexutil.Encode(v.Blob.BlockRootSlice()),
				Index:         fmt.Sprintf("%d", v.Blob.Index),
				Slot:          fmt.Sprintf("%d", v.Blob.Slot()),
				VersionedHash: versionedHash.String(),
				KzgCommitment: hexutil.Encode(v.Blob.KzgCommitment),
			})
		}, nil
	case *operation.AttesterSlashingReceivedData:
		slashing, ok := v.AttesterSlashing.(*eth.AttesterSlashing)
		if !ok {
			return nil, errors.Wrapf(errUnhandledEventData, "Unexpected type %T for the .AttesterSlashing field of AttesterSlashingReceivedData", v.AttesterSlashing)
		}
		return func() io.Reader {
			return jsonMarshalReader(eventName, structs.AttesterSlashingFromConsensus(slashing))
		}, nil
	case *operation.ProposerSlashingReceivedData:
		return func() io.Reader {
			return jsonMarshalReader(eventName, structs.ProposerSlashingFromConsensus(v.ProposerSlashing))
		}, nil
	case *ethpb.EventFinalizedCheckpoint:
		return func() io.Reader {
			return jsonMarshalReader(eventName, structs.FinalizedCheckpointEventFromV1(v))
		}, nil
	case *ethpbv2.LightClientFinalityUpdateWithVersion:
		cv, err := structs.LightClientFinalityUpdateFromConsensus(v.Data)
		if err != nil {
			return nil, errors.Wrap(err, "LightClientFinalityUpdateWithVersion event conversion failure")
		}
		ev := &structs.LightClientFinalityUpdateEvent{
			Version: version.String(int(v.Version)),
			Data:    cv,
		}
		return func() io.Reader {
			return jsonMarshalReader(eventName, ev)
		}, nil
	case *ethpbv2.LightClientOptimisticUpdateWithVersion:
		cv, err := structs.LightClientOptimisticUpdateFromConsensus(v.Data)
		if err != nil {
			return nil, errors.Wrap(err, "LightClientOptimisticUpdateWithVersion event conversion failure")
		}
		ev := &structs.LightClientOptimisticUpdateEvent{
			Version: version.String(int(v.Version)),
			Data:    cv,
		}
		return func() io.Reader {
			return jsonMarshalReader(eventName, ev)
		}, nil
	case *ethpb.EventChainReorg:
		return func() io.Reader {
			return jsonMarshalReader(eventName, structs.EventChainReorgFromV1(v))
		}, nil
	case *statefeed.BlockProcessedData:
		blockRoot, err := v.SignedBlock.Block().HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not compute block root for BlockProcessedData state feed event")
		}
		return func() io.Reader {
			blk := &structs.BlockEvent{
				Slot:                fmt.Sprintf("%d", v.Slot),
				Block:               hexutil.Encode(blockRoot[:]),
				ExecutionOptimistic: v.Optimistic,
			}
			return jsonMarshalReader(eventName, blk)
		}, nil
	default:
		return nil, errors.Wrapf(errUnhandledEventData, "event data type %T unsupported", v)
	}
}

// This event stream is intended to be used by builders and relays.
// Parent fields are based on state at N_{current_slot}, while the rest of fields are based on state of N_{current_slot + 1}
func (s *Server) currentPayloadAttributes(ctx context.Context) (lazyReader, error) {
	headRoot, err := s.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head root")
	}
	st, err := s.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state")
	}
	// advance the head state
	headState, err := transition.ProcessSlotsIfPossible(ctx, st, s.ChainInfoFetcher.CurrentSlot()+1)
	if err != nil {
		return nil, errors.Wrap(err, "could not advance head state")
	}

	headBlock, err := s.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head block")
	}

	headPayload, err := headBlock.Block().Body().Execution()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload")
	}

	t, err := slots.ToTime(headState.GenesisTime(), headState.Slot())
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state slot time")
	}

	prevRando, err := helpers.RandaoMix(headState, chaintime.CurrentEpoch(headState))
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state randao mix")
	}

	proposerIndex, err := helpers.BeaconProposerIndex(ctx, headState)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state proposer index")
	}
	feeRecipient := params.BeaconConfig().DefaultFeeRecipient.Bytes()
	tValidator, exists := s.TrackedValidatorsCache.Validator(proposerIndex)
	if exists {
		feeRecipient = tValidator.FeeRecipient[:]
	}
	var attributes interface{}
	switch headState.Version() {
	case version.Bellatrix:
		attributes = &structs.PayloadAttributesV1{
			Timestamp:             fmt.Sprintf("%d", t.Unix()),
			PrevRandao:            hexutil.Encode(prevRando),
			SuggestedFeeRecipient: hexutil.Encode(feeRecipient),
		}
	case version.Capella:
		withdrawals, _, err := headState.ExpectedWithdrawals()
		if err != nil {
			return nil, errors.Wrap(err, "could not get head state expected withdrawals")
		}
		attributes = &structs.PayloadAttributesV2{
			Timestamp:             fmt.Sprintf("%d", t.Unix()),
			PrevRandao:            hexutil.Encode(prevRando),
			SuggestedFeeRecipient: hexutil.Encode(feeRecipient),
			Withdrawals:           structs.WithdrawalsFromConsensus(withdrawals),
		}
	case version.Deneb, version.Electra:
		withdrawals, _, err := headState.ExpectedWithdrawals()
		if err != nil {
			return nil, errors.Wrap(err, "could not get head state expected withdrawals")
		}
		parentRoot, err := headBlock.Block().HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not get head block root")
		}
		attributes = &structs.PayloadAttributesV3{
			Timestamp:             fmt.Sprintf("%d", t.Unix()),
			PrevRandao:            hexutil.Encode(prevRando),
			SuggestedFeeRecipient: hexutil.Encode(feeRecipient),
			Withdrawals:           structs.WithdrawalsFromConsensus(withdrawals),
			ParentBeaconBlockRoot: hexutil.Encode(parentRoot[:]),
		}
	default:
		return nil, errors.Wrapf(err, "Payload version %s is not supported", version.String(headState.Version()))
	}

	attributesBytes, err := json.Marshal(attributes)
	if err != nil {
		return nil, errors.Wrap(err, "errors marshaling payload attributes to json")
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
		return nil, errors.Wrap(err, "errors marshaling payload attributes event data to json")
	}
	return func() io.Reader {
		return jsonMarshalReader(PayloadAttributesTopic, &structs.PayloadAttributesEvent{
			Version: version.String(headState.Version()),
			Data:    eventDataBytes,
		})
	}, nil
}
