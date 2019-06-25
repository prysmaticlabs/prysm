package spectest

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	log "github.com/sirupsen/logrus"
)

func TestBlockProcessingYaml(t *testing.T) {
	ctx := context.Background()

	file, err := ioutil.ReadFile("sanity_blocks_minimal.yaml")
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
		stateConfig := state.DefaultConfig()

		for _, b := range testCase.Blocks {
			serializedObj, err := json.Marshal(b)
			if err != nil {
				t.Fatal(err)
			}
			protoBlock := &pb.BeaconBlock{}
			if err := jsonpb.Unmarshal(bytes.NewReader(serializedObj), protoBlock); err != nil {
				t.Fatal(err)
			}
			postState, err = state.ExecuteStateTransition(ctx, preState, protoBlock, stateConfig)
			if err != nil {
				t.Fatal(err)
			}
		}

		if !reflect.DeepEqual(postState, testCase.Post) {
			t.Error("Failed")
		}
	}
}
