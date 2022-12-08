package beacon_api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestGetRestJsonResponse_Valid(t *testing.T) {
	const endpoint = "/example/rest/api/endpoint"

	genesisJson := &rpcmiddleware.GenesisResponseJson{
		Data: &rpcmiddleware.GenesisResponse_GenesisJson{
			GenesisTime:           "123",
			GenesisValidatorsRoot: "0x456",
			GenesisForkVersion:    "0x789",
		},
	}

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

	responseJson := &rpcmiddleware.GenesisResponseJson{}
	_, err := jsonRestHandler.GetRestJsonResponse(endpoint+"?arg1=abc&arg2=def", responseJson)
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
			responseJson: &rpcmiddleware.GenesisResponseJson{},
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
			responseJson: &rpcmiddleware.GenesisResponseJson{},
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
			responseJson: &rpcmiddleware.GenesisResponseJson{},
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
			responseJson: &rpcmiddleware.GenesisResponseJson{},
		},
		{
			name:                 "bad error json formatting",
			funcHandler:          invalidJsonErrHandler,
			expectedErrorMessage: "failed to decode error json",
			timeout:              time.Second * 5,
			responseJson:         &rpcmiddleware.GenesisResponseJson{},
		},
		{
			name:                 "bad response json formatting",
			funcHandler:          invalidJsonResponseHandler,
			expectedErrorMessage: "failed to decode response json",
			timeout:              time.Second * 5,
			responseJson:         &rpcmiddleware.GenesisResponseJson{},
		},
		{
			name:                 "timeout",
			funcHandler:          httpErrorJsonHandler(http.StatusNotFound, "Not found"),
			expectedErrorMessage: "failed to query REST API",
			timeout:              1,
			responseJson:         &rpcmiddleware.GenesisResponseJson{},
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
			errorJson, err := jsonRestHandler.GetRestJsonResponse(endpoint, testCase.responseJson)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
			assert.DeepEqual(t, testCase.expectedErrorJson, errorJson)
		})
	}
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
