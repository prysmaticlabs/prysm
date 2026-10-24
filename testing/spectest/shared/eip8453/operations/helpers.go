package operations

import (
	"context"
	"path"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	common "github.com/OffchainLabs/prysm/v7/testing/spectest/shared/common/operations"
	"github.com/OffchainLabs/prysm/v7/testing/spectest/utils"
	"github.com/OffchainLabs/prysm/v7/testing/util"
	"github.com/golang/snappy"
)

func sszToState(b []byte) (state.BeaconState, error) {
	base := &ethpb.BeaconStateEIP8453{}
	if err := base.UnmarshalSSZ(b); err != nil {
		return nil, err
	}
	return state_native.InitializeFromProtoUnsafeEIP8453(base)
}

func blockWithBody(body *ethpb.BeaconBlockBodyEIP8453) (interfaces.SignedBeaconBlock, error) {
	return blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockEIP8453{Block: &ethpb.BeaconBlockEIP8453{Body: body}})
}

// runOperationTest decodes the handler's input into a block body, or uses an empty body when the handler has no input.
func runOperationTest(t *testing.T, config, handler string, body func([]byte) (*ethpb.BeaconBlockBodyEIP8453, error), op func(context.Context, state.BeaconState, interfaces.ReadOnlyBeaconBlockBody) error) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "eip8453", path.Join("operations", handler, "pyspec_tests"))
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			b := &ethpb.BeaconBlockBodyEIP8453{}
			if body != nil {
				file, err := util.BazelFileBytes(folderPath, handler+".ssz_snappy")
				require.NoError(t, err)
				ssz, err := snappy.Decode(nil /* dst */, file)
				require.NoError(t, err, "Failed to decompress")
				b, err = body(ssz)
				require.NoError(t, err)
			}
			blk, err := blockWithBody(b)
			require.NoError(t, err)
			common.RunBlockOperationTest(t, folderPath, blk, sszToState, func(ctx context.Context, s state.BeaconState, blk interfaces.ReadOnlySignedBeaconBlock) (state.BeaconState, error) {
				return s, op(ctx, s, blk.Block().Body())
			})
		})
	}
}
