// This binary is a simple ssz deserializer REST API endpoint for the testnet
// frontend to decode the deposit data for the user.
// This should be removed after https://github.com/prysmaticlabs/prysm-testnet-site/issues/37
// is resolved.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/api/decodeDepositData", decodeDepositData)

	log.Println("Starting on port 4000")
	if err := http.ListenAndServe(":4000", mux); err != nil {
		log.Fatalf("Failed to start server %v", err)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type deserializationRequest struct {
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

	requestData := &deserializationRequest{}
	if err := json.Unmarshal(data, requestData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Unable to unmarshal JSON: %v\n", err)
		return
	}

	if len(requestData.Data) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	encodedData, err := hex.DecodeString(requestData.Data[2:])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err = fmt.Fprintf(w, "Failed to decode hex string")
		if err != nil {
			log.Printf("Failed to write data to client: %v\n", err)
		}
		return
	}

	di := &pb.DepositInput{}

	if err := ssz.Decode(bytes.NewReader(encodedData), di); err != nil {
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		log.Printf("Failed to write data to client: %v\n", err)
	}
}
