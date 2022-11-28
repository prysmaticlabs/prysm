//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
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
	resp, httpError, err := validatorClient.getGenesis()
	assert.NoError(t, err)
	assert.Equal(t, (*apimiddleware.DefaultErrorJson)(nil), httpError)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "1234", resp.Data.GenesisTime)
	assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", resp.Data.GenesisValidatorsRoot)
}

func TestGetGenesis_NilData(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(nil))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, httpError, err := validatorClient.getGenesis()
	assert.Equal(t, (*apimiddleware.DefaultErrorJson)(nil), httpError)
	assert.ErrorContains(t, "GenesisResponseJson.Data is nil", err)
}

func TestGetGenesis_InvalidJsonGenesis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("foo"))
		require.NoError(t, err)
	}))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, httpError, err := validatorClient.getGenesis()
	assert.Equal(t, (*apimiddleware.DefaultErrorJson)(nil), httpError)
	assert.ErrorContains(t, "failed to decode response body genesis json", err)
}

func TestGetGenesis_InvalidJsonError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(invalidJsonErrHandler))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, httpError, err := validatorClient.getGenesis()
	assert.Equal(t, (*apimiddleware.DefaultErrorJson)(nil), httpError)
	assert.ErrorContains(t, "failed to decode response body genesis error json", err)
}

func TestGetGenesis_404Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(notFoundErrHandler))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, httpError, err := validatorClient.getGenesis()
	require.NotNil(t, httpError)
	assert.Equal(t, http.StatusNotFound, httpError.Code)
	assert.Equal(t, "Not found", httpError.Message)
	assert.ErrorContains(t, "error 404: Not found", err)
}

func TestGetGenesis_500Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(internalServerErrHandler))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, httpError, err := validatorClient.getGenesis()
	require.NotNil(t, httpError)
	assert.Equal(t, http.StatusInternalServerError, httpError.Code)
	assert.Equal(t, "Internal server error", httpError.Message)
	assert.ErrorContains(t, "error 500: Internal server error", err)
}

func TestGetGenesis_Timeout(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(&rpcmiddleware.GenesisResponse_GenesisJson{
		GenesisTime:           "1234",
		GenesisValidatorsRoot: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
	}))
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: 1}}
	_, httpError, err := validatorClient.getGenesis()
	assert.Equal(t, (*apimiddleware.DefaultErrorJson)(nil), httpError)
	assert.ErrorContains(t, "failed to query REST API genesis endpoint", err)
}
