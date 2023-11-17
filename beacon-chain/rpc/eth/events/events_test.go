package events

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mockChain "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

type flushableResponseRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flushableResponseRecorder) Flush() {
	f.flushed = true
}

func TestStreamEvents_OperationsEvents(t *testing.T) {
	t.Run("operations", func(t *testing.T) {
		s := &Server{
			StateNotifier:     &mockChain.MockStateNotifier{},
			OperationNotifier: &mockChain.MockOperationNotifier{},
		}

		topics := []string{AttestationTopic, VoluntaryExitTopic, SyncCommitteeContributionTopic, BLSToExecutionChangeTopic, BlobSidecarTopic}
		for i, topic := range topics {
			topics[i] = "topics=" + topic
		}
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com/eth/v1/events?%s", strings.Join(topics, "&")), nil)
		w := &flushableResponseRecorder{
			ResponseRecorder: httptest.NewRecorder(),
		}

		go func() {
			s.StreamEvents(w, request)
		}()
		// wait for initiation of StreamEvents
		time.Sleep(100 * time.Millisecond)
		s.OperationNotifier.OperationFeed().Send(&feed.Event{
			Type: operation.UnaggregatedAttReceived,
			Data: &operation.UnAggregatedAttReceivedData{
				Attestation: util.HydrateAttestation(&eth.Attestation{}),
			},
		})
		s.OperationNotifier.OperationFeed().Send(&feed.Event{
			Type: operation.AggregatedAttReceived,
			Data: &operation.AggregatedAttReceivedData{
				Attestation: &eth.AggregateAttestationAndProof{
					AggregatorIndex: 0,
					Aggregate:       util.HydrateAttestation(&eth.Attestation{}),
					SelectionProof:  make([]byte, 96),
				},
			},
		})
		s.OperationNotifier.OperationFeed().Send(&feed.Event{
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
		})
		s.OperationNotifier.OperationFeed().Send(&feed.Event{
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
		})
		s.OperationNotifier.OperationFeed().Send(&feed.Event{
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
		})
		s.OperationNotifier.OperationFeed().Send(&feed.Event{
			Type: operation.BlobSidecarReceived,
			Data: &operation.BlobSidecarReceivedData{
				Blob: util.HydrateSignedBlobSidecar(&eth.SignedBlobSidecar{}),
			},
		})
		// wait for feed
		time.Sleep(100 * time.Millisecond)
		request.Context().Done()

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NotNil(t, body)
		assert.Equal(t, string(body), operationsResult)
	})
	t.Run("state", func(t *testing.T) {
		s := &Server{
			StateNotifier:     &mockChain.MockStateNotifier{},
			OperationNotifier: &mockChain.MockOperationNotifier{},
		}

		topics := []string{HeadTopic, PayloadAttributesTopic, FinalizedCheckpointTopic, ChainReorgTopic, BlockTopic}
		for i, topic := range topics {
			topics[i] = "topics=" + topic
		}
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com/eth/v1/events?%s", strings.Join(topics, "&")), nil)
		w := &flushableResponseRecorder{
			ResponseRecorder: httptest.NewRecorder(),
		}

		go func() {
			s.StreamEvents(w, request)
		}()
		// wait for initiation of StreamEvents
		time.Sleep(100 * time.Millisecond)
		s.StateNotifier.StateFeed().Send(&feed.Event{
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
		})

		// wait for feed
		time.Sleep(100 * time.Millisecond)
		request.Context().Done()

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NotNil(t, body)
		assert.Equal(t, string(body), operationsResult)
	})
}

const operationsResult = `event: attestation
data: {"aggregation_bits":"0x00","data":{"slot":"0","index":"0","beacon_block_root":"0x0000000000000000000000000000000000000000000000000000000000000000","source":{"epoch":"0","root":"0x0000000000000000000000000000000000000000000000000000000000000000"},"target":{"epoch":"0","root":"0x0000000000000000000000000000000000000000000000000000000000000000"}},"signature":"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"}

event: attestation
data: {"aggregation_bits":"0x00","data":{"slot":"0","index":"0","beacon_block_root":"0x0000000000000000000000000000000000000000000000000000000000000000","source":{"epoch":"0","root":"0x0000000000000000000000000000000000000000000000000000000000000000"},"target":{"epoch":"0","root":"0x0000000000000000000000000000000000000000000000000000000000000000"}},"signature":"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"}

event: voluntary_exit
data: {"message":{"epoch":"0","validator_index":"0"},"signature":"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"}

event: contribution_and_proof
data: {"message":{"aggregator_index":"0","contribution":{"slot":"0","beacon_block_root":"0x0000000000000000000000000000000000000000000000000000000000000000","subcommittee_index":"0","aggregation_bits":"0x00000000000000000000000000000000","signature":"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},"selection_proof":"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},"signature":"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"}

event: bls_to_execution_change
data: {"message":{"validator_index":"0","from_bls_pubkey":"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","to_execution_address":"0x0000000000000000000000000000000000000000"},"signature":"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"}

event: blob_sidecar
data: {"block_root":"0x0000000000000000000000000000000000000000000000000000000000000000","index":"0","slot":"0","kzg_commitment":"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","versioned_hash":"0x01b0761f87b081d5cf10757ccc89f12be355c70e2e29df288b65b30710dcbcd1"}

`
