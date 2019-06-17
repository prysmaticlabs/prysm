package spectest

import (
	"context"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var slotsProcessingTests SanitySlotsMinimal

func TestSlotProcessingYaml(t *testing.T) {
	ctx := context.Background()

	for _, testCase := range slotsProcessingTests.TestCases {
		t.Logf("Description: %s", testCase.Description)

		preState := testCase.Pre
		var postState *pb.BeaconState

		for s:=uint64(0); s < testCase.Slots; s++ {
			postState, err := state.ProcessSlot(ctx, preState)
			if err != nil {
				t.Fatal(err)
			}
		}

		if !reflect.DeepEqual(postState, testCase.Post) {
			t.Error("bleh bleh bleh")
		}
	}
}
