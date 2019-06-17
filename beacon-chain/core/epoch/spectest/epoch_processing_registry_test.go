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

var  registryUpdateTests RegistryUpdatesMinimal

func TestRegistryProcessingYaml(t *testing.T) {
	for _, testCase := range registryUpdateTests.TestCases {
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

		postState, err = epoch.ProcessRegistryUpdates(preState)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(postState, testCase.Post) {
			t.Error("Process registry updates mutated state differently than yaml output")
		}
	}
}
