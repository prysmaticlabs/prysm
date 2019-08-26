package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"log"

	"github.com/prysmaticlabs/prysm/shared/bls"
)

var (
	numKeys    = flag.Int("num-keys", 0, "Number of validator private/withdrawal keys to generate")
	outputJSON = flag.String("output-json", "", "JSON file to write output to")
)

type unencryptedKeysContainer struct {
	Keys []*unencryptedKeys `json:"keys"`
}

type unencryptedKeys struct {
	ValidatorKey  []byte `json:"validator_key"`
	WithdrawalKey []byte `json:"withdrawal_key"`
}

func main() {
	flag.Parse()
	if *numKeys == 0 {
		log.Fatal("Please specify --num-keys to generate")
	}
	if *outputJSON == "" {
		log.Fatal("Please specify an --output-json file to write the unencrypted keys to")
	}

	ctnr := &unencryptedKeysContainer{
		Keys: make([]*unencryptedKeys, *numKeys),
	}
	for i := 0; i < *numKeys; i++ {
		signingKey, err := bls.RandKey(rand.Reader)
		if err != nil {
			log.Fatal(err)
		}
		withdrawalKey, err := bls.RandKey(rand.Reader)
		if err != nil {
			log.Fatal(err)
		}
		ctnr.Keys[i] = &unencryptedKeys{
			ValidatorKey:  signingKey.Marshal(),
			WithdrawalKey: withdrawalKey.Marshal(),
		}
	}

	log.Print(len(ctnr.Keys))

	enc, err := json.Marshal(ctnr)
	if err != nil {
		log.Fatal(err)
	}

	newCont := &unencryptedKeysContainer{}
	if err := json.Unmarshal(enc, newCont); err != nil {
		log.Fatal(err)
	}

	log.Print(len(newCont.Keys))
	//file, err := os.Create(*outputJSON)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer file.Close()
	//
	//n, err := file.Write(enc)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//if n != len(enc) {
	//	log.Fatalf("Failed to write %d bytes to file, wrote %d", len(enc), n)
	//}
}
