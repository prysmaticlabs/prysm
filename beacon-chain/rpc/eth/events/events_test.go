package events

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	mockChain "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	sse "github.com/r3labs/sse/v2"
)

type flushableResponseRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flushableResponseRecorder) Flush() {
	f.flushed = true
}

func NewFlushabaleResponseRecorder() *flushableResponseRecorder {
	return &flushableResponseRecorder{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

func requireAllEventsReceived(t *testing.T, events []*feed.Event, req *topicRequest, s *Server, w *StreamingResponseWriterRecorder) {
	// maxBufferSize param copied from sse lib client code
	sseR := sse.NewEventStreamReader(w.Body(), 1<<24)
	testTimeout := time.NewTimer(60 * time.Second)
	// Wait for the initial write and flush which will write the http header.
	// This is also a signal that the server side has subscribed to the feed,
	// so the setup code below can proceed with writing events.
	select {
	case status := <-w.StatusChan():
		if status != http.StatusOK {
			t.Fatalf("expected status OK, got %d", status)
		}
	case <-testTimeout.C:
		t.Fatal("timed out waiting for events")
	}

	expected := make(map[string]bool)
	for i := range events {
		ev := events[i]
		// serialize the event the same way the server will so that we can compare expectation to results.
		top := topicForEvent(ev)
		eb, err := s.lazyReaderForEvent(context.Background(), ev, req)
		require.NoError(t, err)
		exb, err := io.ReadAll(eb())
		require.NoError(t, err)
		exs := string(exb[0 : len(exb)-2]) // remove trailing double newline

		if topicsForOpsFeed[top] {
			// Send the event on the feed.
			s.OperationNotifier.OperationFeed().Send(ev)
		} else {
			if !topicsForStateFeed[top] {
				t.Fatalf("could not determine feed for topic: %s", top)
			}
			// Send the event on the feed.
			s.StateNotifier.StateFeed().Send(ev)
		}
		expected[exs] = true
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			ev, err := sseR.ReadEvent()
			if err == io.EOF {
				return
			}
			require.NoError(t, err)
			str := string(ev)
			delete(expected, str)
			if len(expected) == 0 {
				return
			}
		}
	}()
	select {
	case <-done:
		break
	case <-testTimeout.C:
		t.Fatal("timed out waiting for events")
	}
	// Clean up the timeout timer.
	if !testTimeout.Stop() {
		<-testTimeout.C
	}
	require.Equal(t, 0, len(expected), "expected events not seen")
}

func (tr *topicRequest) testHttpRequest(_ *testing.T) *http.Request {
	tq := make([]string, 0, len(tr.topics))
	for topic := range tr.topics {
		tq = append(tq, "topics="+topic)
	}
	return httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com/eth/v1/events?%s", strings.Join(tq, "&")), nil)
}

func TestStreamEvents_OperationsEvents(t *testing.T) {
	t.Run("operations", func(t *testing.T) {
		s := &Server{
			StateNotifier:     mockChain.NewSimpleStateNotifier(),
			OperationNotifier: &mockChain.MockOperationNotifier{},
		}

		topics, err := newTopicRequest([]string{
			AttestationTopic,
			VoluntaryExitTopic,
			SyncCommitteeContributionTopic,
			BLSToExecutionChangeTopic,
			BlobSidecarTopic,
			AttesterSlashingTopic,
			ProposerSlashingTopic,
		})
		require.NoError(t, err)
		request := topics.testHttpRequest(t)
		w := NewStreamingResponseWriterRecorder()

		go func() {
			s.StreamEvents(w, request)
		}()

		ro, err := blocks.NewROBlob(util.HydrateBlobSidecar(&eth.BlobSidecar{}))
		require.NoError(t, err)
		vblob := blocks.NewVerifiedROBlob(ro)

		events := []*feed.Event{
			&feed.Event{
				Type: operation.UnaggregatedAttReceived,
				Data: &operation.UnAggregatedAttReceivedData{
					Attestation: util.HydrateAttestation(&eth.Attestation{}),
				},
			},
			&feed.Event{
				Type: operation.AggregatedAttReceived,
				Data: &operation.AggregatedAttReceivedData{
					Attestation: &eth.AggregateAttestationAndProof{
						AggregatorIndex: 0,
						Aggregate:       util.HydrateAttestation(&eth.Attestation{}),
						SelectionProof:  make([]byte, 96),
					},
				},
			},
			&feed.Event{
				Type: operation.ExitReceived,
				Data: &operation.ExitReceivedData{
					Exit: &eth.SignedVoluntaryExit{
						Exit: &eth.VoluntaryExit{
							Epoch:          0,
							ValidatorIndex: 0,
						},
						Signature: make([]byte, 96),
					},
				},
			},
			&feed.Event{
				Type: operation.SyncCommitteeContributionReceived,
				Data: &operation.SyncCommitteeContributionReceivedData{
					Contribution: &eth.SignedContributionAndProof{
						Message: &eth.ContributionAndProof{
							AggregatorIndex: 0,
							Contribution: &eth.SyncCommitteeContribution{
								Slot:              0,
								BlockRoot:         make([]byte, 32),
								SubcommitteeIndex: 0,
								AggregationBits:   make([]byte, 16),
								Signature:         make([]byte, 96),
							},
							SelectionProof: make([]byte, 96),
						},
						Signature: make([]byte, 96),
					},
				},
			},
			&feed.Event{
				Type: operation.BLSToExecutionChangeReceived,
				Data: &operation.BLSToExecutionChangeReceivedData{
					Change: &eth.SignedBLSToExecutionChange{
						Message: &eth.BLSToExecutionChange{
							ValidatorIndex:     0,
							FromBlsPubkey:      make([]byte, 48),
							ToExecutionAddress: make([]byte, 20),
						},
						Signature: make([]byte, 96),
					},
				},
			},
			&feed.Event{
				Type: operation.BlobSidecarReceived,
				Data: &operation.BlobSidecarReceivedData{
					Blob: &vblob,
				},
			},
			&feed.Event{
				Type: operation.AttesterSlashingReceived,
				Data: &operation.AttesterSlashingReceivedData{
					AttesterSlashing: &eth.AttesterSlashing{
						Attestation_1: &eth.IndexedAttestation{
							AttestingIndices: []uint64{0, 1},
							Data: &eth.AttestationData{
								BeaconBlockRoot: make([]byte, fieldparams.RootLength),
								Source: &eth.Checkpoint{
									Root: make([]byte, fieldparams.RootLength),
								},
								Target: &eth.Checkpoint{
									Root: make([]byte, fieldparams.RootLength),
								},
							},
							Signature: make([]byte, fieldparams.BLSSignatureLength),
						},
						Attestation_2: &eth.IndexedAttestation{
							AttestingIndices: []uint64{0, 1},
							Data: &eth.AttestationData{
								BeaconBlockRoot: make([]byte, fieldparams.RootLength),
								Source: &eth.Checkpoint{
									Root: make([]byte, fieldparams.RootLength),
								},
								Target: &eth.Checkpoint{
									Root: make([]byte, fieldparams.RootLength),
								},
							},
							Signature: make([]byte, fieldparams.BLSSignatureLength),
						},
					},
				},
			},
			&feed.Event{
				Type: operation.ProposerSlashingReceived,
				Data: &operation.ProposerSlashingReceivedData{
					ProposerSlashing: &eth.ProposerSlashing{
						Header_1: &eth.SignedBeaconBlockHeader{
							Header: &eth.BeaconBlockHeader{
								ParentRoot: make([]byte, fieldparams.RootLength),
								StateRoot:  make([]byte, fieldparams.RootLength),
								BodyRoot:   make([]byte, fieldparams.RootLength),
							},
							Signature: make([]byte, fieldparams.BLSSignatureLength),
						},
						Header_2: &eth.SignedBeaconBlockHeader{
							Header: &eth.BeaconBlockHeader{
								ParentRoot: make([]byte, fieldparams.RootLength),
								StateRoot:  make([]byte, fieldparams.RootLength),
								BodyRoot:   make([]byte, fieldparams.RootLength),
							},
							Signature: make([]byte, fieldparams.BLSSignatureLength),
						},
					},
				},
			},
		}
		requireAllEventsReceived(t, events, topics, s, w)
	})
	t.Run("state", func(t *testing.T) {
		s := &Server{
			StateNotifier:     mockChain.NewSimpleStateNotifier(),
			OperationNotifier: &mockChain.MockOperationNotifier{},
		}

		topics, err := newTopicRequest([]string{
			HeadTopic,
			FinalizedCheckpointTopic,
			ChainReorgTopic,
			BlockTopic,
		})
		require.NoError(t, err)
		request := topics.testHttpRequest(t)
		w := NewStreamingResponseWriterRecorder()

		go func() {
			s.StreamEvents(w, request)
		}()

		b, err := blocks.NewSignedBeaconBlock(util.HydrateSignedBeaconBlock(&eth.SignedBeaconBlock{}))
		require.NoError(t, err)
		events := []*feed.Event{
			&feed.Event{
				Type: statefeed.BlockProcessed,
				Data: &statefeed.BlockProcessedData{
					Slot:        0,
					BlockRoot:   [32]byte{},
					SignedBlock: b,
					Verified:    true,
					Optimistic:  false,
				},
			},
			&feed.Event{
				Type: statefeed.NewHead,
				Data: &ethpb.EventHead{
					Slot:                      0,
					Block:                     make([]byte, 32),
					State:                     make([]byte, 32),
					EpochTransition:           true,
					PreviousDutyDependentRoot: make([]byte, 32),
					CurrentDutyDependentRoot:  make([]byte, 32),
					ExecutionOptimistic:       false,
				},
			},
			&feed.Event{
				Type: statefeed.Reorg,
				Data: &ethpb.EventChainReorg{
					Slot:                0,
					Depth:               0,
					OldHeadBlock:        make([]byte, 32),
					NewHeadBlock:        make([]byte, 32),
					OldHeadState:        make([]byte, 32),
					NewHeadState:        make([]byte, 32),
					Epoch:               0,
					ExecutionOptimistic: false,
				},
			},
			&feed.Event{
				Type: statefeed.FinalizedCheckpoint,
				Data: &ethpb.EventFinalizedCheckpoint{
					Block:               make([]byte, 32),
					State:               make([]byte, 32),
					Epoch:               0,
					ExecutionOptimistic: false,
				},
			},
		}
		requireAllEventsReceived(t, events, topics, s, w)
	})
	t.Run("payload attributes", func(t *testing.T) {
		type testCase struct {
			name                      string
			getState                  func() state.BeaconState
			getBlock                  func() interfaces.SignedBeaconBlock
			SetTrackedValidatorsCache func(*cache.TrackedValidatorsCache)
		}
		testCases := []testCase{
			{
				name: "bellatrix",
				getState: func() state.BeaconState {
					st, err := util.NewBeaconStateBellatrix()
					require.NoError(t, err)
					return st
				},
				getBlock: func() interfaces.SignedBeaconBlock {
					b, err := blocks.NewSignedBeaconBlock(util.HydrateSignedBeaconBlockBellatrix(&eth.SignedBeaconBlockBellatrix{}))
					require.NoError(t, err)
					return b
				},
			},
			{
				name: "capella",
				getState: func() state.BeaconState {
					st, err := util.NewBeaconStateCapella()
					require.NoError(t, err)
					return st
				},
				getBlock: func() interfaces.SignedBeaconBlock {
					b, err := blocks.NewSignedBeaconBlock(util.HydrateSignedBeaconBlockCapella(&eth.SignedBeaconBlockCapella{}))
					require.NoError(t, err)
					return b
				},
			},
			{
				name: "deneb",
				getState: func() state.BeaconState {
					st, err := util.NewBeaconStateDeneb()
					require.NoError(t, err)
					return st
				},
				getBlock: func() interfaces.SignedBeaconBlock {
					b, err := blocks.NewSignedBeaconBlock(util.HydrateSignedBeaconBlockDeneb(&eth.SignedBeaconBlockDeneb{}))
					require.NoError(t, err)
					return b
				},
			},
			{
				name: "electra",
				getState: func() state.BeaconState {
					st, err := util.NewBeaconStateElectra()
					require.NoError(t, err)
					return st
				},
				getBlock: func() interfaces.SignedBeaconBlock {
					b, err := blocks.NewSignedBeaconBlock(util.HydrateSignedBeaconBlockElectra(&eth.SignedBeaconBlockElectra{}))
					require.NoError(t, err)
					return b
				},
				SetTrackedValidatorsCache: func(c *cache.TrackedValidatorsCache) {
					c.Set(cache.TrackedValidator{
						Active:       true,
						Index:        0,
						FeeRecipient: primitives.ExecutionAddress(common.HexToAddress("0xd2DBd02e4efe087d7d195de828b9Dd25f19A89C9").Bytes()),
					})
				},
			},
		}
		for _, tc := range testCases {
			st := tc.getState()
			v := &eth.Validator{ExitEpoch: math.MaxUint64}
			require.NoError(t, st.SetValidators([]*eth.Validator{v}))
			currentSlot := primitives.Slot(0)
			// to avoid slot processing
			require.NoError(t, st.SetSlot(currentSlot+1))
			b := tc.getBlock()
			mockChainService := &mockChain.ChainService{
				Root:  make([]byte, 32),
				State: st,
				Block: b,
				Slot:  &currentSlot,
			}

			s := &Server{
				StateNotifier:          mockChain.NewSimpleStateNotifier(),
				OperationNotifier:      &mockChain.MockOperationNotifier{},
				HeadFetcher:            mockChainService,
				ChainInfoFetcher:       mockChainService,
				TrackedValidatorsCache: cache.NewTrackedValidatorsCache(),
			}
			if tc.SetTrackedValidatorsCache != nil {
				tc.SetTrackedValidatorsCache(s.TrackedValidatorsCache)
			}
			topics, err := newTopicRequest([]string{PayloadAttributesTopic})
			require.NoError(t, err)
			request := topics.testHttpRequest(t)
			w := NewStreamingResponseWriterRecorder()
			events := []*feed.Event{&feed.Event{Type: statefeed.MissedSlot}}

			go func() {
				s.StreamEvents(w, request)
			}()
			requireAllEventsReceived(t, events, topics, s, w)
		}
	})
}
