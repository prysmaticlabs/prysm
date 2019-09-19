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
	startIndex = flag.Uint64("start-index", 0, "Start index for the determinstic keygen algorithm")
	outputJSON = flag.String("output-json", "", "JSON file to write output to")
	overwrite  = flag.Bool("overwrite", false, "If the key file exists, it will be overwritten")
)

// UnencryptedKeysContainer defines the structure of the unecrypted key JSON file.
type UnencryptedKeysContainer struct {
	Keys []*UnencryptedKeys `json:"keys"`
}

// UnencryptedKeys is the inner struct of the JSON file.
type UnencryptedKeys struct {
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

	ctnr := generateUnencryptedKeys(*startIndex)
	if err := SaveUnencryptedKeysToFile(file, ctnr); err != nil {
		log.Fatal(err)
	}
}

func generateUnencryptedKeys(startIndex uint64) *UnencryptedKeysContainer {
	ctnr := &UnencryptedKeysContainer{
		Keys: make([]*UnencryptedKeys, *numKeys),
	}

	sks, _, err := interop.DeterministicallyGenerateKeys(startIndex, uint64(*numKeys))

	if err != nil {
		panic(err)
	}

	for i, sk := range sks {
		ctnr.Keys[i] = &UnencryptedKeys{
			ValidatorKey:  sk.Marshal(),
			WithdrawalKey: sk.Marshal(),
		}
	}
	return ctnr
}

// SaveUnencryptedKeysToFile JSON encodes the container and writes to the writer.
func SaveUnencryptedKeysToFile(w io.Writer, ctnr *UnencryptedKeysContainer) error {
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
