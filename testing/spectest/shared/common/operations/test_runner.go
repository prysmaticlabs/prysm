package operations

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/golang/snappy"
	"github.com/google/go-cmp/cmp"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

type blockWithSSZObject func([]byte) (interfaces.SignedBeaconBlock, error)
type BlockOperation func(context.Context, state.BeaconState, interfaces.ReadOnlySignedBeaconBlock) (state.BeaconState, error)
type ProcessBlock func(context.Context, state.BeaconState, interfaces.ReadOnlyBeaconBlock) (state.BeaconState, error)
type SSZToState func([]byte) (state.BeaconState, error)

// RunBlockOperationTest takes in the prestate and the beacon block body, processes it through the
// passed in block operation function and checks the post state with the expected post state.
func RunBlockOperationTest(
	t *testing.T,
	folderPath string,
	wsb interfaces.SignedBeaconBlock,
	sszToState SSZToState,
	operationFn BlockOperation,
) {
	preBeaconStateFile, err := util.BazelFileBytes(path.Join(folderPath, "pre.ssz_snappy"))
	require.NoError(t, err)
	preBeaconStateSSZ, err := snappy.Decode(nil /* dst */, preBeaconStateFile)
	require.NoError(t, err, "Failed to decompress")
	preState, err := sszToState(preBeaconStateSSZ)
	require.NoError(t, err)

	// If the post.ssz is not present, it means the test should fail on our end.
	postSSZFilepath, err := bazel.Runfile(path.Join(folderPath, "post.ssz_snappy"))
	postSSZExists := true
	if err != nil && strings.Contains(err.Error(), "could not locate file") {
		postSSZExists = false
	} else if err != nil {
		t.Fatal(err)
	}

	helpers.ClearCache()
	beaconState, err := operationFn(context.Background(), preState, wsb)
	if postSSZExists {
		require.NoError(t, err)
		comparePostState(t, postSSZFilepath, sszToState, preState, beaconState)
	} else {
		// Note: This doesn't test anything worthwhile. It essentially tests
		// that *any* error has occurred, not any specific error.
		if err == nil {
			t.Fatal("Did not fail when expected")
		}
		t.Logf("Expected failure; failure reason = %v", err)
		return
	}
}

func comparePostState(t *testing.T, postSSZFilepath string, sszToState SSZToState, want state.BeaconState, got state.BeaconState) {
	postBeaconStateFile, err := os.ReadFile(postSSZFilepath) // #nosec G304
	require.NoError(t, err)
	postBeaconStateSSZ, err := snappy.Decode(nil /* dst */, postBeaconStateFile)
	require.NoError(t, err, "Failed to decompress")
	postBeaconState, err := sszToState(postBeaconStateSSZ)
	require.NoError(t, err)
	postBeaconStatePb, ok := postBeaconState.ToProtoUnsafe().(proto.Message)
	require.Equal(t, true, ok, "post beacon state did not return a proto.Message")
	pbState, ok := want.ToProtoUnsafe().(proto.Message)
	require.Equal(t, true, ok, "beacon state did not return a proto.Message")
	if !proto.Equal(pbState, postBeaconStatePb) {
		t.Log(cmp.Diff(postBeaconStatePb, pbState, protocmp.Transform()))
		t.Fatal("Post state does not match expected")
	}
}
