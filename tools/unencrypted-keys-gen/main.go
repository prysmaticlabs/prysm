package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/prysmaticlabs/prysm/shared/interop"
)

var (
	numKeys    = flag.Int("num-keys", 0, "Number of validator private/withdrawal keys to generate")
	outputJSON = flag.String("output-json", "", "JSON file to write output to")
	overwrite  = flag.Bool("overwrite", false, "If the key file exists, it will be overwritten")
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

	if !*overwrite {
		if _, err := os.Stat(*outputJSON); err == nil {
			log.Fatal("The file exists. Use a different file name or the --overwrite flag")
		}
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

	ctnr := generateUnencryptedKeys()
	if err := saveUnencryptedKeysToFile(file, ctnr); err != nil {
		log.Fatal(err)
	}
}

func generateUnencryptedKeys() *unencryptedKeysContainer {
	ctnr := &unencryptedKeysContainer{
		Keys: make([]*unencryptedKeys, *numKeys),
	}

	sks, _, err := interop.DeterministicallyGenerateKeys(0 /*startIndex*/, uint64(*numKeys))

	if err != nil {
		panic(err)
	}

	for i, sk := range sks {
		ctnr.Keys[i] = &unencryptedKeys{
			ValidatorKey:  sk.Marshal(),
			WithdrawalKey: sk.Marshal(),
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
