// This binary is a simple rest API endpoint to calculate
// the ENR value of a node given its private key,ip address and port.
package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/crypto"
	_ "go.uber.org/automaxprocs"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/enr", respondToEnrReq)

	log.Println("Starting on port 4000")
	if err := http.ListenAndServe(":4000", mux); err != nil {
		log.Fatalf("Failed to start server %v", err)
	}
}

type deserializationRequest struct {
	PrivateKey string `json:"privateKey"`
	Ipv4       string `json:"ipv4"`
	UdpPort    int    `json:"udpPort"`
}

func respondToEnrReq(w http.ResponseWriter, r *http.Request) {
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

	decodedKey, err := crypto.ConfigDecodeKey(requestData.PrivateKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Unable to decode private key: %v\n", err)
		return
	}

	privatekey, err := crypto.UnmarshalPrivateKey(decodedKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Unable to unmarshal private key: %v\n", err)
		return
	}

	ecdsaPrivKey := (*ecdsa.PrivateKey)((*btcec.PrivateKey)(privatekey.(*crypto.Secp256k1PrivateKey)))

	if net.ParseIP(requestData.Ipv4).To4() == nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Invalid ipv4 address given: %v\n", err)
		return
	}

	if requestData.UdpPort == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Invalid udp port given: %v\n", err)
		return
	}

	db, err := enode.OpenDB("")
	defer db.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not open node's peer database: %v\n", err)
		return
	}

	localNode := enode.NewLocalNode(db, ecdsaPrivKey)
	ipEntry := enr.IP(net.ParseIP(requestData.Ipv4))
	udpEntry := enr.UDP(requestData.UdpPort)
	localNode.Set(ipEntry)
	localNode.Set(udpEntry)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(localNode.Node().String()))
	if err != nil {
		log.Printf("Failed to write data to client: %v\n", err)
	}
}
