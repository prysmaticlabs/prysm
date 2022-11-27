//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestGetGenesis_ValidGenesis(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(&rpcmiddleware.GenesisResponse_GenesisJson{
		GenesisTime:           "1234",
		GenesisValidatorsRoot: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
	}))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	resp, err := validatorClient.getGenesis()
	assert.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "1234", resp.Data.GenesisTime)
	assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", resp.Data.GenesisValidatorsRoot)
}

func TestGetGenesis_NilData(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(nil))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err := validatorClient.getGenesis()
	assert.ErrorContains(t, "GenesisResponseJson.Data is nil", err)
}

func TestGetGenesis_InvalidJsonGenesis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("foo"))
		require.NoError(t, err)
	}))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err := validatorClient.getGenesis()
	assert.ErrorContains(t, "failed to decode response body genesis json", err)
}

func TestGetGenesis_InvalidJsonError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(invalidJsonErrHandler))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err := validatorClient.getGenesis()
	assert.ErrorContains(t, "failed to decode response body genesis error json", err)
}

func TestGetGenesis_Timeout(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(&rpcmiddleware.GenesisResponse_GenesisJson{
		GenesisTime:           "1234",
		GenesisValidatorsRoot: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
	}))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: 1}}
	_, err := validatorClient.getGenesis()
	assert.ErrorContains(t, "failed to query REST API genesis endpoint", err)
}
