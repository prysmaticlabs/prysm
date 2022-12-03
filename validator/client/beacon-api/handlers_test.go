//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/json"
	"net/http"

	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
)

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

func invalidJsonErrHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, err := w.Write([]byte("foo"))
	if err != nil {
		panic(err)
	}
}
