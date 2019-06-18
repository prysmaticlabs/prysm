package spectest

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/jsonpb"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestSlotProcessingYaml(t *testing.T) {
	ctx := context.Background()

	file, err := ioutil.ReadFile("sanity_slots_minimal.yaml")
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &SanitySlotsMinimalTest{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	for _, testCase := range s.TestCases {
		preState := &pb.BeaconState{}
		t.Logf("Description: %s", testCase.Description)
		b, err := json.Marshal(testCase.Pre)
		if err != nil {
			t.Fatal(err)
		}
		err = jsonpb.Unmarshal(bytes.NewReader(b), preState)
		if err != nil {
			t.Fatal(err)
		}

		var postState *pb.BeaconState

		postState, err = state.ProcessSlots(ctx, preState, preState.Slot + testCase.Slots)
			if err != nil {
				t.Fatal(err)
			}

		genPostState := &pb.BeaconState{}
		b, err = json.Marshal(testCase.Post)
		if err != nil {
			t.Fatal(err)
		}
		err = jsonpb.Unmarshal(bytes.NewReader(b), genPostState)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(postState)
		t.Log(genPostState)
		if !reflect.DeepEqual(postState, genPostState) {
			t.Error("Failed")
		}
	}
}
