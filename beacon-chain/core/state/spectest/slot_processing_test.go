package spectest

import (
	"context"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

func runSlotProcessingTests(t *testing.T, config string) {
	if err := spectest.SetConfig(config); err != nil {
		t.Fatal(err)
	}

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "sanity/slots/pyspec_tests")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearAllCaches()
			preBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
			if err != nil {
				t.Fatal(err)
			}
			beaconState := &pb.BeaconState{}
			if err := ssz.Unmarshal(preBeaconStateFile, beaconState); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			file, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "slots.yaml")
			if err != nil {
				t.Fatal(err)
			}
			fileStr := string(file)
			slotsCount, err := strconv.Atoi(fileStr[:len(fileStr)-5])
			if err != nil {
				t.Fatal(err)
			}

			postBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "post.ssz")
			if err != nil {
				t.Fatal(err)
			}
			postBeaconState := &pb.BeaconState{}
			if err := ssz.Unmarshal(postBeaconStateFile, postBeaconState); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			postState, err := state.ProcessSlots(context.Background(), beaconState, beaconState.Slot+uint64(slotsCount))
			if err != nil {
				t.Fatal(err)
			}

			if !proto.Equal(postState, postBeaconState) {
				diff, _ := messagediff.PrettyDiff(beaconState, postBeaconState)
				t.Log(diff)
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
