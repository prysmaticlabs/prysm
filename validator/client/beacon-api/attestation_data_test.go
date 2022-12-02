//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

const attestationDataEndpoint = "/eth/v1/validator/attestation_data"

func TestGetAttestationData_ValidAttestation(t *testing.T) {
	expectedSlot := uint64(5)
	expectedCommitteeIndex := uint64(6)
	expectedBeaconBlockRoot := "0x0636045df9bdda3ab96592cf5389032c8ec3977f911e2b53509b348dfe164d4d"
	expectedSourceEpoch := uint64(7)
	expectedSourceRoot := "0xd4bcbdefc8156e85247681086e8050e5d2d5d1bf076a25f6decd99250f3a378d"
	expectedTargetEpoch := uint64(8)
	expectedTargetRoot := "0x246590e8e4c2a9bd13cc776ecc7025bc432219f076e80b27267b8fa0456dc821"

	// Mock the attestation_data endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(attestationDataEndpoint, createAttestationDataHandler(&rpcmiddleware.ProduceAttestationDataResponseJson{
		Data: &rpcmiddleware.AttestationDataJson{
			Slot:            strconv.FormatUint(expectedSlot, 10),
			CommitteeIndex:  strconv.FormatUint(expectedCommitteeIndex, 10),
			BeaconBlockRoot: expectedBeaconBlockRoot,
			Source: &rpcmiddleware.CheckpointJson{
				Epoch: strconv.FormatUint(expectedSourceEpoch, 10),
				Root:  expectedSourceRoot,
			},
			Target: &rpcmiddleware.CheckpointJson{
				Epoch: strconv.FormatUint(expectedTargetEpoch, 10),
				Root:  expectedTargetRoot,
			},
		},
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{
		url:        server.URL,
		httpClient: http.Client{Timeout: time.Second * 5},
	}

	resp, err := validatorClient.getAttestationData(types.Slot(expectedSlot), types.CommitteeIndex(expectedCommitteeIndex))
	assert.NoError(t, err)

	require.NotNil(t, resp)
	assert.Equal(t, expectedBeaconBlockRoot, hexutil.Encode(resp.BeaconBlockRoot))
	assert.Equal(t, expectedCommitteeIndex, uint64(resp.CommitteeIndex))
	assert.Equal(t, expectedSlot, uint64(resp.Slot))

	require.NotNil(t, resp.Source)
	assert.Equal(t, expectedSourceEpoch, uint64(resp.Source.Epoch))
	assert.Equal(t, expectedSourceRoot, hexutil.Encode(resp.Source.Root))

	require.NotNil(t, resp.Target)
	assert.Equal(t, expectedTargetEpoch, uint64(resp.Target.Epoch))
	assert.Equal(t, expectedTargetRoot, hexutil.Encode(resp.Target.Root))
}

func TestGetAttestationData_InvalidData(t *testing.T) {
	testCases := []struct {
		name                 string
		generateData         func() *rpcmiddleware.ProduceAttestationDataResponseJson
		expectedErrorMessage string
	}{
		{
			name: "nil attestation data",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				return &rpcmiddleware.ProduceAttestationDataResponseJson{
					Data: nil,
				}
			},
			expectedErrorMessage: "attestation data is nil",
		},
		{
			name: "invalid committee index",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				attestation := generateValidAttestation(1, 2)
				attestation.Data.CommitteeIndex = "foo"
				return attestation
			},
			expectedErrorMessage: "failed to parse attestation committee index: foo",
		},
		{
			name: "invalid block root",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				attestation := generateValidAttestation(1, 2)
				attestation.Data.BeaconBlockRoot = "foo"
				return attestation
			},
			expectedErrorMessage: "invalid beacon block root: foo",
		},
		{
			name: "invalid slot",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				attestation := generateValidAttestation(1, 2)
				attestation.Data.Slot = "foo"
				return attestation
			},
			expectedErrorMessage: "failed to parse attestation slot: foo",
		},
		{
			name: "nil source",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				attestation := generateValidAttestation(1, 2)
				attestation.Data.Source = nil
				return attestation
			},
			expectedErrorMessage: "attestation source is nil",
		},
		{
			name: "invalid source epoch",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				attestation := generateValidAttestation(1, 2)
				attestation.Data.Source.Epoch = "foo"
				return attestation
			},
			expectedErrorMessage: "failed to parse attestation source epoch: foo",
		},
		{
			name: "invalid source root",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				attestation := generateValidAttestation(1, 2)
				attestation.Data.Source.Root = "foo"
				return attestation
			},
			expectedErrorMessage: "invalid attestation source root: foo",
		},
		{
			name: "nil target",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				attestation := generateValidAttestation(1, 2)
				attestation.Data.Target = nil
				return attestation
			},
			expectedErrorMessage: "attestation target is nil",
		},
		{
			name: "invalid target epoch",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				attestation := generateValidAttestation(1, 2)
				attestation.Data.Target.Epoch = "foo"
				return attestation
			},
			expectedErrorMessage: "failed to parse attestation target epoch: foo",
		},
		{
			name: "invalid target root",
			generateData: func() *rpcmiddleware.ProduceAttestationDataResponseJson {
				attestation := generateValidAttestation(1, 2)
				attestation.Data.Target.Root = "foo"
				return attestation
			},
			expectedErrorMessage: "invalid attestation target root: foo",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Mock the attestation_data endpoint
			mux := http.NewServeMux()
			mux.HandleFunc(attestationDataEndpoint, createAttestationDataHandler(testCase.generateData()))
			server := httptest.NewServer(mux)
			defer server.Close()

			validatorClient := &beaconApiValidatorClient{
				url:        server.URL,
				httpClient: http.Client{Timeout: time.Second * 5},
			}

			_, err := validatorClient.getAttestationData(1, 2)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

func TestGetAttestationData_InvalidAttestationJson(t *testing.T) {
	// Mock the attestation_data endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(attestationDataEndpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("foo"))
		require.NoError(t, err)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{
		url:        server.URL,
		httpClient: http.Client{Timeout: time.Second * 5},
	}

	_, err := validatorClient.getAttestationData(1, 2)
	assert.ErrorContains(t, "failed to decode response body attestation_data json", err)
}

func TestGetAttestationData_InvalidErrorJson(t *testing.T) {
	// Mock the attestation_data endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(attestationDataEndpoint, http.HandlerFunc(invalidJsonErrHandler))
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{
		url:        server.URL,
		httpClient: http.Client{Timeout: time.Second * 5},
	}

	_, err := validatorClient.getAttestationData(1, 2)
	assert.ErrorContains(t, "failed to decode response body attestation_data error json", err)
}

func TestGetAttestationData_404Error(t *testing.T) {
	// Mock the attestation_data endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(attestationDataEndpoint, http.HandlerFunc(notFoundErrHandler))
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{
		url:        server.URL,
		httpClient: http.Client{Timeout: time.Second * 5},
	}

	_, err := validatorClient.getAttestationData(1, 2)
	assert.ErrorContains(t, "error 404: Not found", err)
}

func TestGetAttestationData_500Error(t *testing.T) {
	// Mock the attestation_data endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(attestationDataEndpoint, http.HandlerFunc(internalServerErrHandler))
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{
		url:        server.URL,
		httpClient: http.Client{Timeout: time.Second * 5},
	}

	_, err := validatorClient.getAttestationData(1, 2)
	assert.ErrorContains(t, "error 500: Internal server error", err)
}

func TestGetAttestationData_Timeout(t *testing.T) {
	// Mock the attestation_data endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(attestationDataEndpoint, http.HandlerFunc(internalServerErrHandler))
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{
		url:        server.URL,
		httpClient: http.Client{Timeout: 1},
	}

	_, err := validatorClient.getAttestationData(1, 2)
	assert.ErrorContains(t, "failed to query REST API /eth/v1/validator/attestation_data endpoint", err)
	assert.ErrorContains(t, "context deadline exceeded", err)
}

func createAttestationDataHandler(data *rpcmiddleware.ProduceAttestationDataResponseJson) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		marshalledResponse, err := json.Marshal(data)
		if err != nil {
			panic(err)
		}

		_, err = w.Write(marshalledResponse)
		if err != nil {
			panic(err)
		}
	})
}

func generateValidAttestation(slot uint64, committeeIndex uint64) *rpcmiddleware.ProduceAttestationDataResponseJson {
	return &rpcmiddleware.ProduceAttestationDataResponseJson{
		Data: &rpcmiddleware.AttestationDataJson{
			Slot:            strconv.FormatUint(slot, 10),
			CommitteeIndex:  strconv.FormatUint(committeeIndex, 10),
			BeaconBlockRoot: "0x5ecf3bff35e39d5f75476d42950d549f81fa93038c46b6652ae89ae1f7ad834f",
			Source: &rpcmiddleware.CheckpointJson{
				Epoch: "3",
				Root:  "0x9023c9e64f23c1d451d5073c641f5f69597c2ad7d82f6f16e67d703e0ce5db8b",
			},
			Target: &rpcmiddleware.CheckpointJson{
				Epoch: "4",
				Root:  "0xb154d46803b15b458ca822466547b054bc124338c6ee1d9c433dcde8c4457cca",
			},
		},
	}
}
