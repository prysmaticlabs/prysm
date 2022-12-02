//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
)

func TestBeaconApiValidatorClient_GetAttestationDataNilInput(t *testing.T) {
	validatorClient := beaconApiValidatorClient{}
	_, err := validatorClient.GetAttestationData(context.Background(), nil)
	assert.ErrorContains(t, "GetAttestationData received nil argument `in`", err)
}

// Make sure that GetAttestationData() returns the same thing as the internal getAttestationData()
func TestBeaconApiValidatorClient_GetAttestationDataValid(t *testing.T) {
	const slot = types.Slot(1)
	const committeeIndex = types.CommitteeIndex(2)

	mux := http.NewServeMux()
	mux.HandleFunc(attestationDataEndpoint, createAttestationDataHandler(generateValidAttestation(uint64(slot), uint64(committeeIndex))))
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := beaconApiValidatorClient{
		url:        server.URL,
		httpClient: http.Client{Timeout: time.Second * 5},
	}

	expectedResp, expectedErr := validatorClient.getAttestationData(slot, committeeIndex)

	resp, err := validatorClient.GetAttestationData(
		context.Background(),
		&ethpb.AttestationDataRequest{Slot: slot, CommitteeIndex: committeeIndex},
	)

	assert.DeepEqual(t, expectedErr, err)
	assert.DeepEqual(t, expectedResp, resp)
}

func TestBeaconApiValidatorClient_GetAttestationDataError(t *testing.T) {
	const slot = types.Slot(1)
	const committeeIndex = types.CommitteeIndex(2)

	mux := http.NewServeMux()
	mux.HandleFunc(attestationDataEndpoint, notFoundErrHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := beaconApiValidatorClient{
		url:        server.URL,
		httpClient: http.Client{Timeout: time.Second * 5},
	}

	expectedResp, expectedErr := validatorClient.getAttestationData(slot, committeeIndex)

	resp, err := validatorClient.GetAttestationData(
		context.Background(),
		&ethpb.AttestationDataRequest{Slot: slot, CommitteeIndex: committeeIndex},
	)

	assert.ErrorContains(t, expectedErr.Error(), err)
	assert.DeepEqual(t, expectedResp, resp)
}
