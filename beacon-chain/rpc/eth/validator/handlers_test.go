package validator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	mockChain "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	p2pmock "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestGetAggregateAttestation(t *testing.T) {
	root1 := bytesutil.PadTo([]byte("root1"), 32)
	sig1 := bytesutil.PadTo([]byte("sig1"), fieldparams.BLSSignatureLength)
	attSlot1 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root1,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
		},
		Signature: sig1,
	}
	root21 := bytesutil.PadTo([]byte("root2_1"), 32)
	sig21 := bytesutil.PadTo([]byte("sig2_1"), fieldparams.BLSSignatureLength)
	attslot21 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  2,
			BeaconBlockRoot: root21,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root21,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root21,
			},
		},
		Signature: sig21,
	}
	root22 := bytesutil.PadTo([]byte("root2_2"), 32)
	sig22 := bytesutil.PadTo([]byte("sig2_2"), fieldparams.BLSSignatureLength)
	attslot22 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root22,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root22,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root22,
			},
		},
		Signature: sig22,
	}
	root33 := bytesutil.PadTo([]byte("root3_3"), 32)
	sig33 := bls.NewAggregateSignature().Marshal()
	attslot33 := &ethpbalpha.Attestation{
		AggregationBits: []byte{1, 0, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root33,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root33,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root33,
			},
		},
		Signature: sig33,
	}
	pool := attestations.NewPool()
	err := pool.SaveAggregatedAttestations([]*ethpbalpha.Attestation{attSlot1, attslot21, attslot22})
	assert.NoError(t, err)
	s := &Server{
		AttestationsPool: pool,
	}

	t.Run("ok", func(t *testing.T) {
		reqRoot, err := attslot22.Data.HashTreeRoot()
		require.NoError(t, err)
		attDataRoot := hexutil.Encode(reqRoot[:])
		url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &AggregateAttestationResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		assert.DeepEqual(t, "0x00010101", resp.Data.AggregationBits)
		assert.DeepEqual(t, hexutil.Encode(sig22), resp.Data.Signature)
		assert.Equal(t, "2", resp.Data.Data.Slot)
		assert.Equal(t, "3", resp.Data.Data.CommitteeIndex)
		assert.DeepEqual(t, hexutil.Encode(root22), resp.Data.Data.BeaconBlockRoot)
		require.NotNil(t, resp.Data.Data.Source)
		assert.Equal(t, "1", resp.Data.Data.Source.Epoch)
		assert.DeepEqual(t, hexutil.Encode(root22), resp.Data.Data.Source.Root)
		require.NotNil(t, resp.Data.Data.Target)
		assert.Equal(t, "1", resp.Data.Data.Target.Epoch)
		assert.DeepEqual(t, hexutil.Encode(root22), resp.Data.Data.Target.Root)
	})

	t.Run("aggregate beforehand", func(t *testing.T) {
		err = s.AttestationsPool.SaveUnaggregatedAttestation(attslot33)
		require.NoError(t, err)
		newAtt := ethpbalpha.CopyAttestation(attslot33)
		newAtt.AggregationBits = []byte{0, 1, 0, 1}
		err = s.AttestationsPool.SaveUnaggregatedAttestation(newAtt)
		require.NoError(t, err)

		reqRoot, err := attslot33.Data.HashTreeRoot()
		require.NoError(t, err)
		attDataRoot := hexutil.Encode(reqRoot[:])
		url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &AggregateAttestationResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		assert.DeepEqual(t, "0x01010001", resp.Data.AggregationBits)
	})
	t.Run("no matching attestation", func(t *testing.T) {
		attDataRoot := hexutil.Encode(bytesutil.PadTo([]byte("foo"), 32))
		url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusNotFound, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusNotFound, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No matching attestation found"))
	})
	t.Run("no attestation_data_root provided", func(t *testing.T) {
		url := "http://example.com?slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Attestation data root is required"))
	})
	t.Run("invalid attestation_data_root provided", func(t *testing.T) {
		url := "http://example.com?attestation_data_root=foo&slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Attestation data root is invalid"))
	})
	t.Run("no slot provided", func(t *testing.T) {
		attDataRoot := hexutil.Encode(bytesutil.PadTo([]byte("foo"), 32))
		url := "http://example.com?attestation_data_root=" + attDataRoot
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Slot is required"))
	})
	t.Run("invalid slot provided", func(t *testing.T) {
		attDataRoot := hexutil.Encode(bytesutil.PadTo([]byte("foo"), 32))
		url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=foo"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.GetAggregateAttestation(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Slot is invalid"))
	})
}

func TestGetAggregateAttestation_SameSlotAndRoot_ReturnMostAggregationBits(t *testing.T) {
	root := bytesutil.PadTo([]byte("root"), 32)
	sig := bytesutil.PadTo([]byte("sig"), fieldparams.BLSSignatureLength)
	att1 := &ethpbalpha.Attestation{
		AggregationBits: []byte{3, 0, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}
	att2 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 3, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}
	pool := attestations.NewPool()
	err := pool.SaveAggregatedAttestations([]*ethpbalpha.Attestation{att1, att2})
	assert.NoError(t, err)
	s := &Server{
		AttestationsPool: pool,
	}
	reqRoot, err := att1.Data.HashTreeRoot()
	require.NoError(t, err)
	attDataRoot := hexutil.Encode(reqRoot[:])
	url := "http://example.com?attestation_data_root=" + attDataRoot + "&slot=1"
	request := httptest.NewRequest(http.MethodGet, url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetAggregateAttestation(writer, request)
	assert.Equal(t, http.StatusOK, writer.Code)
	resp := &AggregateAttestationResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	require.NotNil(t, resp)
	assert.DeepEqual(t, "0x03000001", resp.Data.AggregationBits)
}

func TestSubmitContributionAndProofs(t *testing.T) {
	s := &Server{
		OperationNotifier: (&mockChain.ChainService{}).OperationNotifier(),
	}

	t.Run("single", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		s.Broadcaster = broadcaster
		s.SyncCommitteePool = synccommittee.NewStore()

		var body bytes.Buffer
		_, err := body.WriteString(singleContribution)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, 1, len(broadcaster.BroadcastMessages))
		contributions, err := s.SyncCommitteePool.SyncCommitteeContributions(1)
		require.NoError(t, err)
		assert.Equal(t, 1, len(contributions))
	})

	t.Run("multiple", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		s.Broadcaster = broadcaster
		s.SyncCommitteePool = synccommittee.NewStore()

		var body bytes.Buffer
		_, err := body.WriteString(multipleContributions)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		assert.Equal(t, 2, len(broadcaster.BroadcastMessages))
		contributions, err := s.SyncCommitteePool.SyncCommitteeContributions(1)
		require.NoError(t, err)
		assert.Equal(t, 2, len(contributions))
	})

	t.Run("invalid", func(t *testing.T) {
		s.SyncCommitteePool = synccommittee.NewStore()

		var body bytes.Buffer
		_, err := body.WriteString(invalidContribution)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
	})

	t.Run("no body", func(t *testing.T) {
		s.SyncCommitteePool = synccommittee.NewStore()

		var body bytes.Buffer
		_, err := body.WriteString(invalidContribution)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitContributionAndProofs(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &http2.DefaultErrorJson{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
}

/*func TestSubmitAggregateAndProofs(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	c := params.BeaconNetworkConfig()
	c.MaximumGossipClockDisparity = time.Hour
	params.OverrideBeaconNetworkConfig(c)
	root := bytesutil.PadTo([]byte("root"), 32)
	sig := bytesutil.PadTo([]byte("sig"), fieldparams.BLSSignatureLength)
	proof := bytesutil.PadTo([]byte("proof"), fieldparams.BLSSignatureLength)
	att := &ethpbv1.Attestation{
		AggregationBits: []byte{0, 1},
		Data: &ethpbv1.AttestationData{
			Slot:            1,
			Index:           1,
			BeaconBlockRoot: root,
			Source: &ethpbv1.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
			Target: &ethpbv1.Checkpoint{
				Epoch: 1,
				Root:  root,
			},
		},
		Signature: sig,
	}

	t.Run("OK", func(t *testing.T) {
		chainSlot := primitives.Slot(0)
		chain := &mockChain.ChainService{
			Genesis: time.Now(), Slot: &chainSlot,
		}
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			TimeFetcher: chain,
			Broadcaster: broadcaster,
		}

		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  proof,
					},
					Signature: sig,
				},
			},
		}

		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, true, broadcaster.BroadcastCalled)
	})

	t.Run("nil aggregate", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}

		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				nil,
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed aggregate request can't be nil", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)

		req = &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message:   nil,
					Signature: sig,
				},
			},
		}
		_, err = vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed aggregate request can't be nil", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)

		req = &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate: &ethpbv1.Attestation{
							AggregationBits: []byte{0, 1},
							Data:            nil,
							Signature:       sig,
						},
						SelectionProof: proof,
					},
					Signature: sig,
				},
			},
		}
		_, err = vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed aggregate request can't be nil", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("zero signature", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}
		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  proof,
					},
					Signature: make([]byte, 96),
				},
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed signatures can't be zero hashes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("zero proof", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}
		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  make([]byte, 96),
					},
					Signature: sig,
				},
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed signatures can't be zero hashes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("zero message signature", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}
		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate: &ethpbv1.Attestation{
							AggregationBits: []byte{0, 1},
							Data: &ethpbv1.AttestationData{
								Slot:            1,
								Index:           1,
								BeaconBlockRoot: root,
								Source: &ethpbv1.Checkpoint{
									Epoch: 1,
									Root:  root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: 1,
									Root:  root,
								},
							},
							Signature: make([]byte, 96),
						},
						SelectionProof: proof,
					},
					Signature: sig,
				},
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Signed signatures can't be zero hashes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("wrong signature length", func(t *testing.T) {
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			Broadcaster: broadcaster,
		}

		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  proof,
					},
					Signature: make([]byte, 99),
				},
			},
		}
		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Incorrect signature length. Expected "+strconv.Itoa(96)+" bytes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)

		req = &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate: &ethpbv1.Attestation{
							AggregationBits: []byte{0, 1},
							Data: &ethpbv1.AttestationData{
								Slot:            1,
								Index:           1,
								BeaconBlockRoot: root,
								Source: &ethpbv1.Checkpoint{
									Epoch: 1,
									Root:  root,
								},
								Target: &ethpbv1.Checkpoint{
									Epoch: 1,
									Root:  root,
								},
							},
							Signature: make([]byte, 99),
						},
						SelectionProof: proof,
					},
					Signature: sig,
				},
			},
		}
		_, err = vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Incorrect signature length. Expected "+strconv.Itoa(96)+" bytes", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})

	t.Run("invalid attestation time", func(t *testing.T) {
		chainSlot := primitives.Slot(0)
		chain := &mockChain.ChainService{
			Genesis: time.Now().Add(time.Hour * 2), Slot: &chainSlot,
		}
		broadcaster := &p2pmock.MockBroadcaster{}
		vs := Server{
			TimeFetcher: chain,
			Broadcaster: broadcaster,
		}

		req := &ethpbv1.SubmitAggregateAndProofsRequest{
			Data: []*ethpbv1.SignedAggregateAttestationAndProof{
				{
					Message: &ethpbv1.AggregateAttestationAndProof{
						AggregatorIndex: 1,
						Aggregate:       att,
						SelectionProof:  proof,
					},
					Signature: sig,
				},
			},
		}

		_, err := vs.SubmitAggregateAndProofs(ctx, req)
		require.NotNil(t, err)
		assert.ErrorContains(t, "Attestation slot is no longer valid from current time", err)
		assert.Equal(t, false, broadcaster.BroadcastCalled)
	})
}*/

const (
	singleContribution = `[
  {
    "message": {
      "aggregator_index": "1",
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
      "contribution": {
        "slot": "1",
        "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "subcommittee_index": "1",
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
      }
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
]`
	multipleContributions = `[
  {
    "message": {
      "aggregator_index": "1",
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
      "contribution": {
        "slot": "1",
        "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "subcommittee_index": "1",
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
      }
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  },
  {
    "message": {
      "aggregator_index": "1",
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
      "contribution": {
        "slot": "1",
        "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "subcommittee_index": "1",
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
      }
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
]`
	// aggregator_index is invalid
	invalidContribution = `[
  {
    "message": {
      "aggregator_index": "foo",
      "selection_proof": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
      "contribution": {
        "slot": "1",
        "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
        "subcommittee_index": "1",
        "aggregation_bits": "0x01",
        "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
      }
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
]`
)
