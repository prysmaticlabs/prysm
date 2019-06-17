package spectest

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var crosslinksTests CrosslinksMinimal

func TestCrosslinksProcessingYaml(t *testing.T) {
	for _, testCase := range crosslinksTests.TestCases {
		t.Logf("Description: %s", testCase.Description)
		b, err := json.Marshal(testCase.Pre)
		if err != nil {
			t.Fatal(err)
		}
		var preState *pb.BeaconState
		var postState *pb.BeaconState

		err = jsonpb.Unmarshal(bytes.NewReader(b), preState)
		if err != nil {
			t.Fatal(err)
		}

		postState, err = epoch.ProcessCrosslinks(preState)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(postState, testCase.Post) {
			t.Error("Process crosslinks mutated state differently than yaml output")
		}
	}
}
