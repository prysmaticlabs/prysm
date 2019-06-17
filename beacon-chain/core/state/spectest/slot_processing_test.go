package spectest

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var slotsProcessingTests SanitySlotsMinimal

func TestSlotProcessingYaml(t *testing.T) {
	ctx := context.Background()

	for _, testCase := range slotsProcessingTests.TestCases {
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
