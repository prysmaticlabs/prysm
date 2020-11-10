package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// GenesisValidator struct representing JSON input we can accept to generate
// a genesis state from, containing a validator's full deposit data as a hex string.
type GenesisValidator struct {
	DepositData string `json:"deposit_data"`
}

var (
	validatorJSONInput = flag.String(
		"validator-json-file",
		"",
		"Path to JSON file formatted as a list of hex public keys and their corresponding deposit data as hex"+
			" such as [ { public_key: '0x1', deposit_data: '0x2' }, ... ]"+
			" this file will be used for generating a genesis state from a list of specified validator public keys",
	)
	numValidators    = flag.Int("num-validators", 0, "Number of validators to deterministically generate in the generated genesis state")
	useMainnetConfig = flag.Bool("mainnet-config", false, "Select whether genesis state should be generated with mainnet or minimal (default) params")
	genesisTime      = flag.Uint64("genesis-time", 0, "Unix timestamp used as the genesis time in the generated genesis state (defaults to now)")
	sszOutputFile    = flag.String("output-ssz", "", "Output filename of the SSZ marshaling of the generated genesis state")
	yamlOutputFile   = flag.String("output-yaml", "", "Output filename of the YAML marshaling of the generated genesis state")
	jsonOutputFile   = flag.String("output-json", "", "Output filename of the JSON marshaling of the generated genesis state")
)

func main() {
	flag.Parse()
	if *genesisTime == 0 {
		log.Print("No --genesis-time specified, defaulting to now")
	}
	if *sszOutputFile == "" && *yamlOutputFile == "" && *jsonOutputFile == "" {
		log.Println("Expected --output-ssz, --output-yaml, or --output-json to have been provided, received nil")
		return
	}
	if !*useMainnetConfig {
		params.OverrideBeaconConfig(params.MinimalSpecConfig())
	}
	var genesisState *pb.BeaconState
	var err error
	if *validatorJSONInput != "" {
		inputFile := *validatorJSONInput
		expanded, err := fileutil.ExpandPath(inputFile)
		if err != nil {
			log.Printf("Could not expand file path %s: %v", inputFile, err)
			return
		}
		inputJSON, err := os.Open(expanded)
		if err != nil {
			log.Printf("Could not open JSON file for reading: %v", err)
			return
		}
		defer func() {
			if err := inputJSON.Close(); err != nil {
				log.Printf("Could not close file %s: %v", inputFile, err)
			}
		}()
		log.Printf("Generating genesis state from input JSON deposit data %s", inputFile)
		genesisState, err = genesisStateFromJSONValidators(inputJSON, *genesisTime)
		if err != nil {
			log.Printf("Could not generate genesis beacon state: %v", err)
			return
		}
	} else {
		if *numValidators == 0 {
			log.Println("Expected --num-validators to have been provided, received 0")
			return
		}
		// If no JSON input is specified, we create the state deterministically from interop keys.
		genesisState, _, err = interop.GenerateGenesisState(*genesisTime, uint64(*numValidators))
		if err != nil {
			log.Printf("Could not generate genesis beacon state: %v", err)
			return
		}
	}

	if *sszOutputFile != "" {
		encodedState, err := genesisState.MarshalSSZ()
		if err != nil {
			log.Printf("Could not ssz marshal the genesis beacon state: %v", err)
			return
		}
		if err := fileutil.WriteFile(*sszOutputFile, encodedState); err != nil {
			log.Printf("Could not write encoded genesis beacon state to file: %v", err)
			return
		}
		log.Printf("Done writing to %s", *sszOutputFile)
	}
	if *yamlOutputFile != "" {
		encodedState, err := yaml.Marshal(genesisState)
		if err != nil {
			log.Printf("Could not yaml marshal the genesis beacon state: %v", err)
			return
		}
		if err := fileutil.WriteFile(*yamlOutputFile, encodedState); err != nil {
			log.Printf("Could not write encoded genesis beacon state to file: %v", err)
			return
		}
		log.Printf("Done writing to %s", *yamlOutputFile)
	}
	if *jsonOutputFile != "" {
		encodedState, err := json.Marshal(genesisState)
		if err != nil {
			log.Printf("Could not json marshal the genesis beacon state: %v", err)
			return
		}
		if err := fileutil.WriteFile(*jsonOutputFile, encodedState); err != nil {
			log.Printf("Could not write encoded genesis beacon state to file: %v", err)
			return
		}
		log.Printf("Done writing to %s", *jsonOutputFile)
	}
}

func genesisStateFromJSONValidators(r io.Reader, genesisTime uint64) (*pb.BeaconState, error) {
	enc, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var validatorsJSON []*GenesisValidator
	if err := json.Unmarshal(enc, &validatorsJSON); err != nil {
		return nil, err
	}
	depositDataList := make([]*ethpb.Deposit_Data, len(validatorsJSON))
	depositDataRoots := make([][]byte, len(validatorsJSON))
	for i, val := range validatorsJSON {
		depositDataString := val.DepositData
		depositDataString = strings.TrimPrefix(depositDataString, "0x")
		depositDataHex, err := hex.DecodeString(depositDataString)
		if err != nil {
			return nil, err
		}
		data := &ethpb.Deposit_Data{}
		if err := data.UnmarshalSSZ(depositDataHex); err != nil {
			return nil, err
		}
		depositDataList[i] = data
		root, err := data.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		depositDataRoots[i] = root[:]
	}
	beaconState, _, err := interop.GenerateGenesisStateFromDepositData(genesisTime, depositDataList, depositDataRoots)
	if err != nil {
		return nil, err
	}
	return beaconState, nil
}
