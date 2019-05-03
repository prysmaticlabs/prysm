// This binary is a simple ssz deserializer REST API endpoint for the testnet
// frontend to decode the deposit data for the user.
// This should be removed after https://github.com/prysmaticlabs/prysm-testnet-site/issues/37
// is resolved.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/decodeDepositData", decodeDepositData)

	log.Println("Starting on port 4000")
	if err := http.ListenAndServe(":4000", mux); err != nil {
		log.Fatalf("Failed to start server %v", err)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type DeserializationRequest struct {
	Data string `json:"data"`
}

func decodeDepositData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Failed to read request body: %v\n", err)
		return
	}

	requestData := &DeserializationRequest{}
	if err := json.Unmarshal(data, requestData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Unable to unmarshal JSON: %v\n", err)
		return
	}

	log.Printf("Decoding %s\n", requestData.Data)

	di, err := helpers.DecodeDepositInput([]byte(requestData.Data))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := fmt.Fprintf(w, "Failed to decode SSZ data: %v", err)
		if err != nil {
			log.Printf("Failed to write data to client: %v\n", err)
		}
		return
	}

	b, err := json.Marshal(di)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Failed to marshal deposit data: %v\n", err)
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		log.Printf("Failed to write data to client: %v\n", err)
	}
}
