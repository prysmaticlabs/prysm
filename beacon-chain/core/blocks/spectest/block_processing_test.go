package spectest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
)

func TestBlockProcessingYaml(t *testing.T) {
	ctx := context.Background()

	file, err := ioutil.ReadFile("sanity_blocks_minimal_formatted.yaml")
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &BlockProcessing{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	log.Infof("Title: %v", s.Title)
	log.Infof("Summary: %v", s.Summary)
	log.Infof("Fork: %v", s.Forks)
	log.Infof("Config: %v", s.Config)

	if err := setConfig(s.Config); err != nil {
		t.Fatalf("Could not set config: %v", err)
	}

	for _, testCase := range s.TestCases {
		t.Logf("Description: %s", testCase.Description)
		b, err := json.Marshal(testCase.Pre)
		if err != nil {
			t.Fatal(err)
		}
		preState := &pb.BeaconState{}
		postState := &pb.BeaconState{}

		err = jsonpb.Unmarshal(bytes.NewReader(b), preState)
		if err != nil {
			t.Fatal(err)
		}

		for _, b := range testCase.Blocks {
			serializedObj, err := json.Marshal(b)
			if err != nil {
				t.Fatal(err)
			}
			protoBlock := &pb.BeaconBlock{}
			if err := jsonpb.Unmarshal(bytes.NewReader(serializedObj), protoBlock); err != nil {
				t.Fatal(err)
			}
			postState, err = state.ProcessBlock(ctx, preState, protoBlock, &state.TransitionConfig{})
			if err != nil {
				t.Fatal(err)
			}
		}

		if !reflect.DeepEqual(postState, testCase.Post) {
			t.Error("Failed")
		}
	}
}

func setConfig(config string) error {
	configDir := "../../config/"
	file, err := ioutil.ReadFile(configDir + config + ".yaml")
	if err != nil {
		return fmt.Errorf("could not find config yaml %v", err)
	}
	decoded := &params.BeaconChainConfig{}
	if err := yaml.Unmarshal(file, decoded); err != nil {
		return fmt.Errorf("could not unmarshal YAML file into config struct: %v", err)
	}
	params.OverrideBeaconConfig(decoded)
	return nil
}
