package finality

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/testing/util"
	"google.golang.org/protobuf/proto"
	"gopkg.in/d4l3k/messagediff.v1"
)

func init() {
	transition.SkipSlotCache.Disable()
}

type Config struct {
	BlocksCount int `json:"blocks_count"`
}

// RunFinalityTest executes finality spec tests.
func RunFinalityTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))

	testFolders, testsFolderPath := utils.TestFolders(t, config, "phase0", "finality/finality/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearCache()
			preBeaconStateFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "pre.ssz_snappy")
			require.NoError(t, err)
			preBeaconStateSSZ, err := snappy.Decode(nil /* dst */, preBeaconStateFile)
			require.NoError(t, err, "Failed to decompress")
			beaconStateBase := &ethpb.BeaconState{}
			require.NoError(t, beaconStateBase.UnmarshalSSZ(preBeaconStateSSZ), "Failed to unmarshal")
			beaconState, err := v1.InitializeFromProto(beaconStateBase)
			require.NoError(t, err)

			file, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "meta.yaml")
			require.NoError(t, err)

			metaYaml := &Config{}
			require.NoError(t, utils.UnmarshalYaml(file, metaYaml), "Failed to Unmarshal")

			var processedState state.BeaconState
			var ok bool
			for i := 0; i < metaYaml.BlocksCount; i++ {
				filename := fmt.Sprintf("blocks_%d.ssz_snappy", i)
				blockFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), filename)
				require.NoError(t, err)
				blockSSZ, err := snappy.Decode(nil /* dst */, blockFile)
				require.NoError(t, err, "Failed to decompress")
				block := &ethpb.SignedBeaconBlock{}
				require.NoError(t, block.UnmarshalSSZ(blockSSZ), "Failed to unmarshal")
				wsb, err := wrapper.WrappedSignedBeaconBlock(block)
				require.NoError(t, err)
				processedState, err = transition.ExecuteStateTransition(context.Background(), beaconState, wsb)
				require.NoError(t, err)
				beaconState, ok = processedState.(*v1.BeaconState)
				require.Equal(t, true, ok)
			}

			postBeaconStateFile, err := util.BazelFileBytes(testsFolderPath, folder.Name(), "post.ssz_snappy")
			require.NoError(t, err)
			postBeaconStateSSZ, err := snappy.Decode(nil /* dst */, postBeaconStateFile)
			require.NoError(t, err, "Failed to decompress")
			postBeaconState := &ethpb.BeaconState{}
			require.NoError(t, postBeaconState.UnmarshalSSZ(postBeaconStateSSZ), "Failed to unmarshal")
			pbState, err := v1.ProtobufBeaconState(beaconState.InnerStateUnsafe())
			require.NoError(t, err)
			if !proto.Equal(pbState, postBeaconState) {
				diff, _ := messagediff.PrettyDiff(beaconState.InnerStateUnsafe(), postBeaconState)
				t.Log(diff)
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
