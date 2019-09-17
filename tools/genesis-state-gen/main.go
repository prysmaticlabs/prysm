package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const (
	blsWithdrawalPrefixByte = byte(0)
)

var (
	numValidators    = flag.Int("num-validators", 0, "Number of validators to deterministically include in the generated genesis state")
	useMainnetConfig = flag.Bool("mainnet-config", false, "Select whether genesis state should be generated with mainnet or minimal (default) params")
	genesisTime      = flag.Uint64("genesis-time", 0, "Unix timestamp used as the genesis time in the generated genesis state")
	sszOutputFile    = flag.String("output-ssz", "", "Output filename of the SSZ marshaling of the generated genesis state")
	yamlOutputFile   = flag.String("output-yaml", "", "Output filename of the YAML marshaling of the generated genesis state")
	jsonOutputFile   = flag.String("output-json", "", "Output filename of the JSON marshaling of the generated genesis state")
)

func main() {
	flag.Parse()
	if *numValidators == 0 {
		log.Fatal("Expected --num-validators to have been provided, received 0")
	}
	if *genesisTime == 0 {
		log.Print("No --genesis-time specified, defaulting to 0 as the unix timestamp")
	}
	if *sszOutputFile == "" && *yamlOutputFile == "" && *jsonOutputFile == "" {
		log.Fatal("Expected --output-ssz, --output-yaml, or --output-json to have been provided, received nil")
	}
	if !*useMainnetConfig {
		params.OverrideBeaconConfig(params.MinimalSpecConfig())
	}

	genesisState, _, err := interop.GenerateGenesisState(*genesisTime, uint64(*numValidators))
	if err != nil {
		log.Fatalf("Could not generate genesis beacon state: %v", err)
	}
	if *sszOutputFile != "" {
		encodedState, err := ssz.Marshal(genesisState)
		if err != nil {
			log.Fatalf("Could not ssz marshal the genesis beacon state: %v", err)
		}
		if err := ioutil.WriteFile(*sszOutputFile, encodedState, 0644); err != nil {
			log.Fatalf("Could not write encoded genesis beacon state to file: %v", err)
		}
		log.Printf("Done writing to %s", *sszOutputFile)
	}
	if *yamlOutputFile != "" {
		encodedState, err := yaml.Marshal(genesisState)
		if err != nil {
			log.Fatalf("Could not yaml marshal the genesis beacon state: %v", err)
		}
		if err := ioutil.WriteFile(*yamlOutputFile, encodedState, 0644); err != nil {
			log.Fatalf("Could not write encoded genesis beacon state to file: %v", err)
		}
		log.Printf("Done writing to %s", *yamlOutputFile)
	}
	if *jsonOutputFile != "" {
		encodedState, err := json.Marshal(genesisState)
		if err != nil {
			log.Fatalf("Could not json marshal the genesis beacon state: %v", err)
		}
		if err := ioutil.WriteFile(*jsonOutputFile, encodedState, 0644); err != nil {
			log.Fatalf("Could not write encoded genesis beacon state to file: %v", err)
		}
		log.Printf("Done writing to %s", *jsonOutputFile)
	}
}
