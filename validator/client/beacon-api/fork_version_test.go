//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestForkVersion_Valid(t *testing.T) {
	altairForkVersion := params.BeaconConfig().AltairForkVersion
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), createForkHandler(&rpcmiddleware.StateForkResponseJson{
		Data: &rpcmiddleware.ForkJson{
			CurrentVersion: hexutil.Encode(altairForkVersion),
		},
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	forkVersion, err := forkVersionProvider.GetForkVersion(altairEpoch)
	assert.NoError(t, err)
	assert.DeepEqual(t, altairForkVersion, forkVersion[:])
}

func TestForkVersion_EpochTooBig(t *testing.T) {
	forkVersionProvider := &beaconApiForkVersionProvider{}
	_, err := forkVersionProvider.GetForkVersion(math.MaxUint64)
	assert.ErrorContains(t, fmt.Sprintf("failed to get the fork version for epoch %d", uint64(math.MaxUint64)), err)
}

func TestForkVersion_NilData(t *testing.T) {
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), createForkHandler(&rpcmiddleware.StateForkResponseJson{
		Data: nil,
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err = forkVersionProvider.GetForkVersion(altairEpoch)
	assert.ErrorContains(t, "state fork data is nil", err)
}

func TestForkVersion_InvalidVersion(t *testing.T) {
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), createForkHandler(&rpcmiddleware.StateForkResponseJson{
		Data: &rpcmiddleware.ForkJson{
			CurrentVersion: "0xzzzzzzzz",
		},
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err = forkVersionProvider.GetForkVersion(altairEpoch)
	assert.ErrorContains(t, "invalid fork version: 0xzzzzzzzz", err)
}

func TestForkVersion_EmptyVersion(t *testing.T) {
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), createForkHandler(&rpcmiddleware.StateForkResponseJson{
		Data: &rpcmiddleware.ForkJson{
			CurrentVersion: "",
		},
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err = forkVersionProvider.GetForkVersion(altairEpoch)
	assert.ErrorContains(t, "invalid fork version: ", err)
}

func TestForkVersion_InvalidJson(t *testing.T) {
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), invalidJsonResultHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err = forkVersionProvider.GetForkVersion(altairEpoch)
	assert.ErrorContains(t, "failed to decode response body state fork json", err)
}

func TestForkVersion_InvalidJsonError(t *testing.T) {
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), invalidJsonErrHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err = forkVersionProvider.GetForkVersion(altairEpoch)
	assert.ErrorContains(t, "failed to decode response body state fork error json", err)
}

func TestForkVersion_InternalServerError(t *testing.T) {
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), internalServerErrHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err = forkVersionProvider.GetForkVersion(altairEpoch)
	assert.ErrorContains(t, "500: Internal server error", err)
}

func TestForkVersion_NotFoundError(t *testing.T) {
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), notFoundErrHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err = forkVersionProvider.GetForkVersion(altairEpoch)
	assert.ErrorContains(t, "404: Not found", err)
}

// This test makes sure that we handle even errors not specified in the Beacon API spec
func TestForkVersion_UnknownError(t *testing.T) {
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), invalidErr999Handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err = forkVersionProvider.GetForkVersion(altairEpoch)
	assert.ErrorContains(t, "999: Invalid error", err)
}

func TestForkVersion_Timeout(t *testing.T) {
	altairForkVersion := params.BeaconConfig().AltairForkVersion
	altairEpoch := params.BeaconConfig().AltairForkEpoch
	altairSlot, err := slots.EpochStart(altairEpoch)
	require.NoError(t, err)

	// Mock the eth/v1/beacon/states/{slot}/fork endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/states/%d/fork", altairSlot), createForkHandler(&rpcmiddleware.StateForkResponseJson{
		Data: &rpcmiddleware.ForkJson{
			CurrentVersion: hexutil.Encode(altairForkVersion),
		},
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	forkVersionProvider := &beaconApiForkVersionProvider{url: server.URL, httpClient: http.Client{Timeout: 1}}
	_, err = forkVersionProvider.GetForkVersion(altairEpoch)
	assert.ErrorContains(t, "failed to query REST API fork endpoint", err)
}

func createForkHandler(data *rpcmiddleware.StateForkResponseJson) http.HandlerFunc {
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
