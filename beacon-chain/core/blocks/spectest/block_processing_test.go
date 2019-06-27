package spectest

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	log "github.com/sirupsen/logrus"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	ctx := context.Background()
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/blocks/sanity_blocks_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}

	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &BlocksMinimal{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	log.Infof("Title: %v", s.Title)
	log.Infof("Summary: %v", s.Summary)
	log.Infof("Fork: %v", s.Forks)
	log.Infof("Config: %v", s.Config)

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatalf("Could not set config: %v", err)
	}

	for _, testCase := range s.TestCases {
		t.Logf("Description: %s", testCase.Description)
		if testCase.Description == "attestation" || testCase.Description == "voluntary_exit" {
			continue
		}
		stateConfig := state.DefaultConfig()

		for _, b := range testCase.Blocks {
			if _, err = state.ExecuteStateTransition(ctx, testCase.Pre, b, stateConfig); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestBlockProcessingMainnetYaml(t *testing.T) {
	ctx := context.Background()
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/blocks/sanity_blocks_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}

	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &BlocksMainnet{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	log.Infof("Title: %v", s.Title)
	log.Infof("Summary: %v", s.Summary)
	log.Infof("Fork: %v", s.Forks)
	log.Infof("Config: %v", s.Config)

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatalf("Could not set config: %v", err)
	}

	for _, testCase := range s.TestCases {
		t.Logf("Description: %s", testCase.Description)
		if testCase.Description == "attestation" || testCase.Description == "voluntary_exit" {
			continue
		}

		stateConfig := state.DefaultConfig()
		for _, b := range testCase.Blocks {
			if _, err = state.ExecuteStateTransition(ctx, testCase.Pre, b, stateConfig); err != nil {
				t.Fatal(err)
			}
		}
	}
}
