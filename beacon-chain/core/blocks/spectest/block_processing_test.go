package spectest

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	// filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/blocks/sanity_blocks_minimal.yaml")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	file, err := ioutil.ReadFile("./sanity_blocks_minimal.yaml")
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &BlocksMinimal{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	t.Logf("Title: %v", s.Title)
	t.Logf("Summary: %v", s.Summary)
	t.Logf("Fork: %v", s.Forks)
	t.Logf("Config: %v", s.Config)

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatalf("Could not set config: %v", err)
	}

	for _, testCase := range s.TestCases {
		if testCase.Description != "voluntary_exit" {
			continue
		}
		helpers.ClearAllCaches()
		currentState := proto.Clone(testCase.Pre).(*pb.BeaconState)
		fmt.Printf("----Description: %s\n", testCase.Description)
		stateConfig := state.DefaultConfig()
		fmt.Printf("Initial state slot: %d\n", currentState.Slot)
		for _, b := range testCase.Blocks {
			parentRoot, err := ssz.SigningRoot(currentState.LatestBlockHeader)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Printf("State latest block header signing root: %#x\n", parentRoot)
			newState, err := state.ExecuteStateTransition(context.Background(), currentState, b, stateConfig)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Printf("Finished state with slot: %d\n", currentState.Slot)
			fmt.Printf("Finished processing block with parent root: %#x\n", b.ParentRoot)
			currentState = proto.Clone(newState).(*pb.BeaconState)
		}
		currentState = nil
	}
}

// func TestBlockProcessingMainnetYaml(t *testing.T) {
// 	ctx := context.Background()
// 	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/blocks/sanity_blocks_mainnet.yaml")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	file, err := ioutil.ReadFile(filepath)
// 	if err != nil {
// 		t.Fatalf("Could not load file %v", err)
// 	}

// 	s := &BlocksMainnet{}
// 	if err := yaml.Unmarshal(file, s); err != nil {
// 		t.Fatalf("Failed to Unmarshal: %v", err)
// 	}

// 	t.Logf("Title: %v", s.Title)
// 	t.Logf("Summary: %v", s.Summary)
// 	t.Logf("Fork: %v", s.Forks)
// 	t.Logf("Config: %v", s.Config)

// 	if err := spectest.SetConfig(s.Config); err != nil {
// 		t.Fatalf("Could not set config: %v", err)
// 	}

// 	for _, testCase := range s.TestCases {
// 		t.Logf("Description: %s", testCase.Description)
// 		if testCase.Description == "attestation" || testCase.Description == "voluntary_exit" {
// 			continue
// 		}

// 		stateConfig := state.DefaultConfig()
// 		for _, b := range testCase.Blocks {
// 			if _, err = state.ExecuteStateTransition(ctx, testCase.Pre, b, stateConfig); err != nil {
// 				t.Fatal(err)
// 			}
// 		}
// 	}
// }
