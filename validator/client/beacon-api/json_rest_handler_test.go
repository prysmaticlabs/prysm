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

	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestGetRestJsonResponse_Valid(t *testing.T) {
	const endpoint = "/example/rest/api/endpoint"

	genesisJson := &beacon.GetGenesisResponse{
		Data: &beacon.Genesis{
			GenesisTime:           "123",
			GenesisValidatorsRoot: "0x456",
			GenesisForkVersion:    "0x789",
		},
	}

	ctx := context.Background()

	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		// Make sure the url parameters match
		assert.Equal(t, "abc", r.URL.Query().Get("arg1"))
		assert.Equal(t, "def", r.URL.Query().Get("arg2"))

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

	responseJson := &beacon.GetGenesisResponse{}
	_, err := jsonRestHandler.GetRestJsonResponse(ctx, endpoint+"?arg1=abc&arg2=def", responseJson)
	assert.NoError(t, err)
	assert.DeepEqual(t, genesisJson, responseJson)
}

func TestGetRestJsonResponse_Error(t *testing.T) {
	const endpoint = "/example/rest/api/endpoint"

	testCases := []struct {
		name                 string
		funcHandler          func(w http.ResponseWriter, r *http.Request)
		expectedErrorJson    *apimiddleware.DefaultErrorJson
		expectedErrorMessage string
		timeout              time.Duration
		responseJson         interface{}
	}{
		{
			name:                 "nil response json",
			funcHandler:          invalidJsonResponseHandler,
			expectedErrorMessage: "responseJson is nil",
			timeout:              time.Second * 5,
			responseJson:         nil,
		},
		{
			name:                 "400 error",
			funcHandler:          httpErrorJsonHandler(http.StatusBadRequest, "Bad request"),
			expectedErrorMessage: "error 400: Bad request",
			expectedErrorJson: &apimiddleware.DefaultErrorJson{
				Code:    http.StatusBadRequest,
				Message: "Bad request",
			},
			timeout:      time.Second * 5,
			responseJson: &beacon.GetGenesisResponse{},
		},
		{
			name:                 "404 error",
			funcHandler:          httpErrorJsonHandler(http.StatusNotFound, "Not found"),
			expectedErrorMessage: "error 404: Not found",
			expectedErrorJson: &apimiddleware.DefaultErrorJson{
				Code:    http.StatusNotFound,
				Message: "Not found",
			},
			timeout:      time.Second * 5,
			responseJson: &beacon.GetGenesisResponse{},
		},
		{
			name:                 "500 error",
			funcHandler:          httpErrorJsonHandler(http.StatusInternalServerError, "Internal server error"),
			expectedErrorMessage: "error 500: Internal server error",
			expectedErrorJson: &apimiddleware.DefaultErrorJson{
				Code:    http.StatusInternalServerError,
				Message: "Internal server error",
			},
			timeout:      time.Second * 5,
			responseJson: &beacon.GetGenesisResponse{},
		},
		{
			name:                 "999 error",
			funcHandler:          httpErrorJsonHandler(999, "Invalid error"),
			expectedErrorMessage: "error 999: Invalid error",
			expectedErrorJson: &apimiddleware.DefaultErrorJson{
				Code:    999,
				Message: "Invalid error",
			},
			timeout:      time.Second * 5,
			responseJson: &beacon.GetGenesisResponse{},
		},
		{
			name:                 "bad error json formatting",
			funcHandler:          invalidJsonErrHandler,
			expectedErrorMessage: "failed to decode error json",
			timeout:              time.Second * 5,
			responseJson:         &beacon.GetGenesisResponse{},
		},
		{
			name:                 "bad response json formatting",
			funcHandler:          invalidJsonResponseHandler,
			expectedErrorMessage: "failed to decode response json",
			timeout:              time.Second * 5,
			responseJson:         &beacon.GetGenesisResponse{},
		},
		{
			name:                 "timeout",
			funcHandler:          httpErrorJsonHandler(http.StatusNotFound, "Not found"),
			expectedErrorMessage: "failed to query REST API",
			timeout:              1,
			responseJson:         &beacon.GetGenesisResponse{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc(endpoint, testCase.funcHandler)
			server := httptest.NewServer(mux)
			defer server.Close()

			ctx := context.Background()

			jsonRestHandler := beaconApiJsonRestHandler{
				httpClient: http.Client{Timeout: testCase.timeout},
				host:       server.URL,
			}
			errorJson, err := jsonRestHandler.GetRestJsonResponse(ctx, endpoint, testCase.responseJson)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
			assert.DeepEqual(t, testCase.expectedErrorJson, errorJson)
		})
	}
}

func TestPostRestJson_Valid(t *testing.T) {
	const endpoint = "/example/rest/api/endpoint"
	dataBytes := []byte{1, 2, 3, 4, 5}

	genesisJson := &beacon.GetGenesisResponse{
		Data: &beacon.Genesis{
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
			responseJson: &beacon.GetGenesisResponse{},
		},
		{
			name:         "empty headers",
			headers:      map[string]string{},
			data:         bytes.NewBuffer(dataBytes),
			responseJson: &beacon.GetGenesisResponse{},
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

			ctx := context.Background()

			jsonRestHandler := beaconApiJsonRestHandler{
				httpClient: http.Client{Timeout: time.Second * 5},
				host:       server.URL,
			}

			_, err := jsonRestHandler.PostRestJson(
				ctx,
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
		responseJson         *beacon.GetGenesisResponse
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
			name:                 "400 error",
			funcHandler:          httpErrorJsonHandler(http.StatusBadRequest, "Bad request"),
			expectedErrorMessage: "error 400: Bad request",
			expectedErrorJson: &apimiddleware.DefaultErrorJson{
				Code:    http.StatusBadRequest,
				Message: "Bad request",
			},
			timeout:      time.Second * 5,
			responseJson: &beacon.GetGenesisResponse{},
			data:         &bytes.Buffer{},
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
			responseJson:         &beacon.GetGenesisResponse{},
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

			ctx := context.Background()

			jsonRestHandler := beaconApiJsonRestHandler{
				httpClient: http.Client{Timeout: testCase.timeout},
				host:       server.URL,
			}

			errorJson, err := jsonRestHandler.PostRestJson(
				ctx,
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

func TestJsonHandler_ContextError(t *testing.T) {
	const endpoint = "/example/rest/api/endpoint"
	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(writer http.ResponseWriter, request *http.Request) {})
	server := httptest.NewServer(mux)
	defer server.Close()

	// Instantiate a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context which results in "context canceled" error.
	cancel()

	jsonRestHandler := beaconApiJsonRestHandler{
		httpClient: http.Client{},
		host:       server.URL,
	}

	_, err := jsonRestHandler.PostRestJson(
		ctx,
		endpoint,
		map[string]string{},
		&bytes.Buffer{},
		nil,
	)

	assert.ErrorContains(t, context.Canceled.Error(), err)

	_, err = jsonRestHandler.GetRestJsonResponse(
		ctx,
		endpoint,
		&beacon.GetGenesisResponse{},
	)

	assert.ErrorContains(t, context.Canceled.Error(), err)
}

func httpErrorJsonHandler(statusCode int, errorMessage string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
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

func invalidJsonErrHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, err := w.Write([]byte("foo"))
	if err != nil {
		panic(err)
	}
}

func invalidJsonResponseHandler(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("foo"))
	if err != nil {
		panic(err)
	}
}
