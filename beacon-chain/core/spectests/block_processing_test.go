package spectests

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
			serializedObj, err := ssz.Marshal(b)
			if err != nil {
				t.Fatal(err)
			}
			protoBlock := &pb.BeaconBlock{}
			if err := ssz.Unmarshal(serializedObj, protoBlock); err != nil {
				t.Fatal(err)
			}
			postState, err = state.ProcessBlock(ctx, preState, protoBlock, nil)
			if err != nil {
				t.Fatal(err)
			}
		}

		if !reflect.DeepEqual(postState, testCase.Post) {
			t.Error("Failed")
		}
	}
}
