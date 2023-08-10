package beacon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbv1alpha1 "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestListAttestations(t *testing.T) {
	att1 := &ethpbv1alpha1.Attestation{
		AggregationBits: []byte{1, 10},
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
			Source: &ethpbv1alpha1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
			},
			Target: &ethpbv1alpha1.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
			},
		},
		Signature: bytesutil.PadTo([]byte("signature1"), 96),
	}
	att2 := &ethpbv1alpha1.Attestation{
		AggregationBits: []byte{1, 10},
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            1,
			CommitteeIndex:  4,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
			Source: &ethpbv1alpha1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
			},
			Target: &ethpbv1alpha1.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
			},
		},
		Signature: bytesutil.PadTo([]byte("signature2"), 96),
	}
	att3 := &ethpbv1alpha1.Attestation{
		AggregationBits: bitfield.NewBitlist(8),
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            2,
			CommitteeIndex:  2,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot3"), 32),
			Source: &ethpbv1alpha1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("sourceroot3"), 32),
			},
			Target: &ethpbv1alpha1.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("targetroot3"), 32),
			},
		},
		Signature: bytesutil.PadTo([]byte("signature3"), 96),
	}
	att4 := &ethpbv1alpha1.Attestation{
		AggregationBits: bitfield.NewBitlist(8),
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            2,
			CommitteeIndex:  4,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot4"), 32),
			Source: &ethpbv1alpha1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("sourceroot4"), 32),
			},
			Target: &ethpbv1alpha1.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("targetroot4"), 32),
			},
		},
		Signature: bytesutil.PadTo([]byte("signature4"), 96),
	}
	s := &Server{
		AttestationsPool: attestations.NewPool(),
	}
	require.NoError(t, s.AttestationsPool.SaveAggregatedAttestations([]*ethpbv1alpha1.Attestation{att1, att2}))
	require.NoError(t, s.AttestationsPool.SaveUnaggregatedAttestations([]*ethpbv1alpha1.Attestation{att3, att4}))

	t.Run("empty request", func(t *testing.T) {
		url := "http://example.com"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.ListAttestations(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &ListAttestationsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		assert.Equal(t, 4, len(resp.Data))
	})
	t.Run("slot request", func(t *testing.T) {
		url := "http://example.com?slot=2"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.ListAttestations(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &ListAttestationsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		assert.Equal(t, 2, len(resp.Data))
		for _, a := range resp.Data {
			assert.Equal(t, "2", a.Data.Slot)
		}
	})
	t.Run("index request", func(t *testing.T) {
		url := "http://example.com?committee_index=4"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.ListAttestations(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &ListAttestationsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		assert.Equal(t, 2, len(resp.Data))
		for _, a := range resp.Data {
			assert.Equal(t, "4", a.Data.CommitteeIndex)
		}
	})
	t.Run("both slot + index request", func(t *testing.T) {
		url := "http://example.com?slot=2&committee_index=4"
		request := httptest.NewRequest(http.MethodGet, url, nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.ListAttestations(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		resp := &ListAttestationsResponse{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
		require.NotNil(t, resp)
		require.NotNil(t, resp.Data)
		assert.Equal(t, 1, len(resp.Data))
		for _, a := range resp.Data {
			assert.Equal(t, "2", a.Data.Slot)
			assert.Equal(t, "4", a.Data.CommitteeIndex)
		}
	})
}
