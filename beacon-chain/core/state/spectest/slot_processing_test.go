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
		t.Logf("Description: %s", testCase.Description)
		b, err := json.Marshal(testCase.Pre)
		if err != nil {
			t.Fatal(err)
		}
		preState := &pb.BeaconState{}
		var postState *pb.BeaconState

		err = jsonpb.Unmarshal(bytes.NewReader(b), preState)
		if err != nil {
			t.Fatal(err)
		}

		for s := uint64(0); s < testCase.Slots; s++ {
			postState, err = state.ProcessSlot(ctx, preState)
			if err != nil {
				t.Fatal(err)
			}
		}

		if !reflect.DeepEqual(postState, testCase.Post) {
			t.Error("Failed")
		}
	}
}
