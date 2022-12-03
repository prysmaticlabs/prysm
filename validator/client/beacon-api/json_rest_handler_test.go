//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
)

func TestGetRestJsonResponse_HttpError(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		errorMessage string
	}{
		{
			name:         "404 error",
			statusCode:   http.StatusNotFound,
			errorMessage: "Not found",
		},
		{
			name:         "500 error",
			statusCode:   http.StatusInternalServerError,
			errorMessage: "Internal server error",
		},
		{
			name:         "999 error",
			statusCode:   999,
			errorMessage: "Invalid error",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/example/rest/api/endpoint", httpErrorJsonHandler(testCase.statusCode, testCase.errorMessage))
			server := httptest.NewServer(mux)
			defer server.Close()

			jsonRestHandler := beaconApiJsonRestHandler{
				httpClient: http.Client{Timeout: time.Second * 5},
				host:       server.URL,
			}
			responseJson := rpcmiddleware.GenesisResponseJson{}
			errorJson, err := jsonRestHandler.GetRestJsonResponse(
				"/example/rest/api/endpoint",
				&responseJson,
			)

			assert.ErrorContains(t, fmt.Sprintf("error %d: %s", testCase.statusCode, testCase.errorMessage), err)
			assert.Equal(t, testCase.statusCode, errorJson.Code)
			assert.Equal(t, testCase.errorMessage, errorJson.Message)
		})
	}
}

func TestGetRestJsonResponse_Timeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/example/rest/api/endpoint", httpErrorJsonHandler(999, "dummy error"))
	server := httptest.NewServer(mux)
	defer server.Close()

	jsonRestHandler := beaconApiJsonRestHandler{
		httpClient: http.Client{Timeout: 1},
		host:       server.URL,
	}
	responseJson := rpcmiddleware.GenesisResponseJson{}
	_, err := jsonRestHandler.GetRestJsonResponse(
		"/example/rest/api/endpoint",
		&responseJson,
	)

	assert.ErrorContains(t, "context deadline exceeded", err)
}
