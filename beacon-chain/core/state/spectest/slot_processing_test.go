package spectest

import (
	"context"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"gopkg.in/d4l3k/messagediff.v1"
)

func init() {
	state.SkipSlotCache.Disable()
}

func runSlotProcessingTests(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "sanity/slots/pyspec_tests")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			preBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
			require.NoError(t, err)
			base := &pb.BeaconState{}
			require.NoError(t, base.UnmarshalSSZ(preBeaconStateFile), "Failed to unmarshal")
			beaconState, err := beaconstate.InitializeFromProto(base)
			require.NoError(t, err)

			file, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "slots.yaml")
			require.NoError(t, err)
			fileStr := string(file)
			slotsCount, err := strconv.Atoi(fileStr[:len(fileStr)-5])
			require.NoError(t, err)

			postBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "post.ssz")
			require.NoError(t, err)
			postBeaconState := &pb.BeaconState{}
			require.NoError(t, postBeaconState.UnmarshalSSZ(postBeaconStateFile), "Failed to unmarshal")
			postState, err := state.ProcessSlots(context.Background(), beaconState, beaconState.Slot()+uint64(slotsCount))
			require.NoError(t, err)

			if !proto.Equal(postState.CloneInnerState(), postBeaconState) {
				diff, _ := messagediff.PrettyDiff(beaconState, postBeaconState)
				t.Fatalf("Post state does not match expected. Diff between states %s", diff)
			}
		})
	}
}
