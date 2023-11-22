package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v4/api"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestGet(t *testing.T) {
	ctx := context.Background()
	const endpoint = "/example/rest/api/endpoint"
	genesisJson := &beacon.GetGenesisResponse{
		Data: &beacon.Genesis{
			GenesisTime:           "123",
			GenesisValidatorsRoot: "0x456",
			GenesisForkVersion:    "0x789",
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		marshalledJson, err := json.Marshal(genesisJson)
		require.NoError(t, err)

		w.Header().Set("Content-Type", api.JsonMediaType)
		_, err = w.Write(marshalledJson)
		require.NoError(t, err)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	jsonRestHandler := beaconApiJsonRestHandler{
		httpClient: http.Client{Timeout: time.Second * 5},
		host:       server.URL,
	}
	resp := &beacon.GetGenesisResponse{}
	errJson, err := jsonRestHandler.Get(ctx, endpoint+"?arg1=abc&arg2=def", resp)
	assert.Equal(t, true, errJson == nil)
	assert.NoError(t, err)
	assert.DeepEqual(t, genesisJson, resp)
}

func TestPost(t *testing.T) {
	ctx := context.Background()
	const endpoint = "/example/rest/api/endpoint"
	dataBytes := []byte{1, 2, 3, 4, 5}
	headers := map[string]string{"foo": "bar"}

	genesisJson := &beacon.GetGenesisResponse{
		Data: &beacon.Genesis{
			GenesisTime:           "123",
			GenesisValidatorsRoot: "0x456",
			GenesisForkVersion:    "0x789",
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		// Make sure the request headers have been set
		assert.Equal(t, "bar", r.Header.Get("foo"))
		assert.Equal(t, api.JsonMediaType, r.Header.Get("Content-Type"))

		// Make sure the data matches
		receivedBytes := make([]byte, len(dataBytes))
		numBytes, err := r.Body.Read(receivedBytes)
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, len(dataBytes), numBytes)
		assert.DeepEqual(t, dataBytes, receivedBytes)

		marshalledJson, err := json.Marshal(genesisJson)
		require.NoError(t, err)

		w.Header().Set("Content-Type", api.JsonMediaType)
		_, err = w.Write(marshalledJson)
		require.NoError(t, err)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	jsonRestHandler := beaconApiJsonRestHandler{
		httpClient: http.Client{Timeout: time.Second * 5},
		host:       server.URL,
	}
	resp := &beacon.GetGenesisResponse{}
	errJson, err := jsonRestHandler.Post(
		ctx,
		endpoint,
		headers,
		bytes.NewBuffer(dataBytes),
		resp,
	)
	assert.Equal(t, true, errJson == nil)
	assert.NoError(t, err)
	assert.DeepEqual(t, genesisJson, resp)
}
