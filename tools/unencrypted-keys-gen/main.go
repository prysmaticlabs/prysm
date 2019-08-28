package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

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

	file, err := os.Create(*outputJSON)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	ctnr := generateUnencryptedKeys(rand.Reader)
	if err := saveUnencryptedKeysToFile(file, ctnr); err != nil {
		log.Fatal(err)
	}
}

func generateUnencryptedKeys(r io.Reader) *unencryptedKeysContainer {
	ctnr := &unencryptedKeysContainer{
		Keys: make([]*unencryptedKeys, *numKeys),
	}
	for i := 0; i < *numKeys; i++ {
		signingKey, err := bls.RandKey(r)
		if err != nil {
			log.Fatal(err)
		}
		withdrawalKey, err := bls.RandKey(r)
		if err != nil {
			log.Fatal(err)
		}
		ctnr.Keys[i] = &unencryptedKeys{
			ValidatorKey:  signingKey.Marshal(),
			WithdrawalKey: withdrawalKey.Marshal(),
		}
	}
	return ctnr
}

func saveUnencryptedKeysToFile(w io.Writer, ctnr *unencryptedKeysContainer) error {
	enc, err := json.Marshal(ctnr)
	if err != nil {
		log.Fatal(err)
	}
	n, err := w.Write(enc)
	if err != nil {
		return err
	}
	if n != len(enc) {
		return fmt.Errorf("failed to write %d bytes to file, wrote %d", len(enc), n)
	}
	return nil
}
