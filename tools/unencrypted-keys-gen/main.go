package main

import (
	"crypto/rand"
	"log"

	"github.com/prysmaticlabs/prysm/shared/bls"
)

type unencryptedKeysContainer struct {
	Keys []*unencryptedKeys `json:"keys"`
}

type unencryptedKeys struct {
	ValidatorKey  []byte `json:"validator_key"`
	WithdrawalKey []byte `json:"withdrawal_key"`
}

func main() {
	privateKey, err := bls.RandKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	publicKey := privateKey.PublicKey()
}
