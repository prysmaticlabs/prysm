package main

import (
	"flag"
	"log"
	"os"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/tools/unencrypted-keys-gen/keygen"
)

var (
	numKeys    = flag.Int("num-keys", 0, "Number of validator private/withdrawal keys to generate")
	startIndex = flag.Uint64("start-index", 0, "Start index for the determinstic keygen algorithm")
	random     = flag.Bool("random", false, "Randomly generate keys")
	outputJSON = flag.String("output-json", "", "JSON file to write output to")
	overwrite  = flag.Bool("overwrite", false, "If the key file exists, it will be overwritten")
)

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

	var ctnr *keygen.UnencryptedKeysContainer
	if *random {
		ctnr = generateRandomKeys(*numKeys)
	} else {
		ctnr = generateUnencryptedKeys(*startIndex)
	}
	if err := keygen.SaveUnencryptedKeysToFile(file, ctnr); err != nil {
		log.Fatal(err)
	}
}

func generateRandomKeys(num int) *keygen.UnencryptedKeysContainer {
	ctnr := &keygen.UnencryptedKeysContainer{
		Keys: make([]*keygen.UnencryptedKeys, num),
	}

	for i := 0; i < num; i++ {
		sk := bls.RandKey()
		ctnr.Keys[i] = &keygen.UnencryptedKeys{
			ValidatorKey:  sk.Marshal(),
			WithdrawalKey: sk.Marshal(),
		}
	}

	return ctnr
}

func generateUnencryptedKeys(startIndex uint64) *keygen.UnencryptedKeysContainer {
	ctnr := &keygen.UnencryptedKeysContainer{
		Keys: make([]*keygen.UnencryptedKeys, *numKeys),
	}

	sks, _, err := interop.DeterministicallyGenerateKeys(startIndex, uint64(*numKeys))

	if err != nil {
		panic(err)
	}

	for i, sk := range sks {
		ctnr.Keys[i] = &keygen.UnencryptedKeys{
			ValidatorKey:  sk.Marshal(),
			WithdrawalKey: sk.Marshal(),
		}
	}
	return ctnr
}
