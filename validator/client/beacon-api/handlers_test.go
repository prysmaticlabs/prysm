//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/json"
	"net/http"

	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
)

func internalServerErrHandler(w http.ResponseWriter, r *http.Request) {
	internalErrorJson := &apimiddleware.DefaultErrorJson{
		Code:    http.StatusInternalServerError,
		Message: "Internal server error",
	}

	marshalledError, err := json.Marshal(internalErrorJson)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusInternalServerError)
	_, err = w.Write(marshalledError)
	if err != nil {
		panic(err)
	}
}

func notFoundErrHandler(w http.ResponseWriter, r *http.Request) {
	internalErrorJson := &apimiddleware.DefaultErrorJson{
		Code:    http.StatusNotFound,
		Message: "Not found",
	}

	marshalledError, err := json.Marshal(internalErrorJson)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusNotFound)
	_, err = w.Write(marshalledError)
	if err != nil {
		panic(err)
	}
}

func invalidErr999Handler(w http.ResponseWriter, r *http.Request) {
	internalErrorJson := &apimiddleware.DefaultErrorJson{
		Code:    999,
		Message: "Invalid error",
	}

	marshalledError, err := json.Marshal(internalErrorJson)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(999)
	_, err = w.Write(marshalledError)
	if err != nil {
		panic(err)
	}
}

func invalidJsonErrHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, err := w.Write([]byte("foo"))
	if err != nil {
		panic(err)
	}
}

func invalidJsonResultHandler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("foo"))
	if err != nil {
		panic(err)
	}
}

func createGenesisHandler(data *rpcmiddleware.GenesisResponse_GenesisJson) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		genesisResponseJson := &rpcmiddleware.GenesisResponseJson{Data: data}
		marshalledResponse, err := json.Marshal(genesisResponseJson)
		if err != nil {
			panic(err)
		}

		_, err = w.Write(marshalledResponse)
		if err != nil {
			panic(err)
		}
	})
}
