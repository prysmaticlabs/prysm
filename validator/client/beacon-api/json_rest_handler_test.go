//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestPostRestJson_Valid(t *testing.T) {
	const endpoint = "/example/rest/api/endpoint"
	dataBytes := []byte{1, 2, 3, 4, 5}

	genesisJson := &rpcmiddleware.GenesisResponseJson{
		Data: &rpcmiddleware.GenesisResponse_GenesisJson{
			GenesisTime:           "123",
			GenesisValidatorsRoot: "0x456",
			GenesisForkVersion:    "0x789",
		},
	}

	testCases := []struct {
		name         string
		headers      map[string]string
		data         *bytes.Buffer
		responseJson interface{}
	}{
		{
			name:         "nil headers",
			headers:      nil,
			data:         bytes.NewBuffer(dataBytes),
			responseJson: &rpcmiddleware.GenesisResponseJson{},
		},
		{
			name:         "empty headers",
			headers:      map[string]string{},
			data:         bytes.NewBuffer(dataBytes),
			responseJson: &rpcmiddleware.GenesisResponseJson{},
		},
		{
			name:         "nil response json",
			headers:      map[string]string{"DummyHeaderKey": "DummyHeaderValue"},
			data:         bytes.NewBuffer(dataBytes),
			responseJson: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
				// Make sure the request headers have been set
				for headerKey, headerValue := range testCase.headers {
					assert.Equal(t, headerValue, r.Header.Get(headerKey))
				}

				// Make sure the data matches
				receivedBytes := make([]byte, len(dataBytes))
				numBytes, err := r.Body.Read(receivedBytes)
				assert.Equal(t, io.EOF, err)
				assert.Equal(t, len(dataBytes), numBytes)
				assert.DeepEqual(t, dataBytes, receivedBytes)

				marshalledJson, err := json.Marshal(genesisJson)
				require.NoError(t, err)

				_, err = w.Write(marshalledJson)
				require.NoError(t, err)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			jsonRestHandler := beaconApiJsonRestHandler{
				httpClient: http.Client{Timeout: time.Second * 5},
				host:       server.URL,
			}

			_, err := jsonRestHandler.PostRestJson(
				endpoint,
				testCase.headers,
				testCase.data,
				testCase.responseJson,
			)

			assert.NoError(t, err)

			if testCase.responseJson != nil {
				assert.DeepEqual(t, genesisJson, testCase.responseJson)
			}
		})
	}
}

func TestPostRestJson_Error(t *testing.T) {
	const endpoint = "/example/rest/api/endpoint"

	testCases := []struct {
		name                 string
		funcHandler          func(w http.ResponseWriter, r *http.Request)
		expectedErrorJson    *apimiddleware.DefaultErrorJson
		expectedErrorMessage string
		timeout              time.Duration
		responseJson         *rpcmiddleware.GenesisResponseJson
		data                 *bytes.Buffer
	}{
		{
			name:                 "nil POST data",
			funcHandler:          httpErrorJsonHandler(http.StatusNotFound, "Not found"),
			expectedErrorMessage: "POST data is nil",
			timeout:              time.Second * 5,
			data:                 nil,
		},
		{
			name:                 "404 error",
			funcHandler:          httpErrorJsonHandler(http.StatusNotFound, "Not found"),
			expectedErrorMessage: "error 404: Not found",
			expectedErrorJson: &apimiddleware.DefaultErrorJson{
				Code:    http.StatusNotFound,
				Message: "Not found",
			},
			timeout: time.Second * 5,
			data:    &bytes.Buffer{},
		},
		{
			name:                 "500 error",
			funcHandler:          httpErrorJsonHandler(http.StatusInternalServerError, "Internal server error"),
			expectedErrorMessage: "error 500: Internal server error",
			expectedErrorJson: &apimiddleware.DefaultErrorJson{
				Code:    http.StatusInternalServerError,
				Message: "Internal server error",
			},
			timeout: time.Second * 5,
			data:    &bytes.Buffer{},
		},
		{
			name:                 "999 error",
			funcHandler:          httpErrorJsonHandler(999, "Invalid error"),
			expectedErrorMessage: "error 999: Invalid error",
			expectedErrorJson: &apimiddleware.DefaultErrorJson{
				Code:    999,
				Message: "Invalid error",
			},
			timeout: time.Second * 5,
			data:    &bytes.Buffer{},
		},
		{
			name:                 "bad error json formatting",
			funcHandler:          invalidJsonErrHandler,
			expectedErrorMessage: "failed to decode error json",
			timeout:              time.Second * 5,
			data:                 &bytes.Buffer{},
		},
		{
			name:                 "bad response json formatting",
			funcHandler:          invalidJsonResponseHandler,
			expectedErrorMessage: "failed to decode response json",
			timeout:              time.Second * 5,
			responseJson:         &rpcmiddleware.GenesisResponseJson{},
			data:                 &bytes.Buffer{},
		},
		{
			name:                 "timeout",
			funcHandler:          httpErrorJsonHandler(http.StatusNotFound, "Not found"),
			expectedErrorMessage: "failed to send POST data to REST endpoint",
			timeout:              1,
			data:                 &bytes.Buffer{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(endpoint, testCase.funcHandler)
			server := httptest.NewServer(mux)
			defer server.Close()

			jsonRestHandler := beaconApiJsonRestHandler{
				httpClient: http.Client{Timeout: testCase.timeout},
				host:       server.URL,
			}
			errorJson, err := jsonRestHandler.PostRestJson(
				endpoint,
				map[string]string{},
				testCase.data,
				testCase.responseJson,
			)

			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
			assert.DeepEqual(t, testCase.expectedErrorJson, errorJson)
		})
	}
}

func httpErrorJsonHandler(statusCode int, errorMessage string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		errorJson := &apimiddleware.DefaultErrorJson{
			Code:    statusCode,
			Message: errorMessage,
		}

		marshalledError, err := json.Marshal(errorJson)
		if err != nil {
			panic(err)
		}

		w.WriteHeader(statusCode)
		_, err = w.Write(marshalledError)
		if err != nil {
			panic(err)
		}
	}
}

func invalidJsonResponseHandler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("foo"))
	if err != nil {
		panic(err)
	}
}
