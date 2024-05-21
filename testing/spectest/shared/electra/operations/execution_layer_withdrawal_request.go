package operations

import (
	"context"
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func RunExecutionLayerWithdrawalRequestTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "electra", "operations/execution_layer_exit/pyspec_tests")
	if len(testFolders) == 0 {
		t.Fatalf("No test folders found for %s/%s/%s", config, "electra", "operations/execution_layer_exit/pyspec_tests")
	}
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			executionLayerExitFile, err := util.BazelFileBytes(folderPath, "execution_layer_exit.ssz_snappy")
			require.NoError(t, err)
			executionLayerExitSSZ, err := snappy.Decode(nil /* dst */, executionLayerExitFile)
			require.NoError(t, err, "Failed to decompress")
			withdrawalRequest := &enginev1.ExecutionLayerWithdrawalRequest{}
			require.NoError(t, withdrawalRequest.UnmarshalSSZ(executionLayerExitSSZ), "Failed to unmarshal")
			body := &ethpb.BeaconBlockBodyElectra{ExecutionPayload: &enginev1.ExecutionPayloadElectra{
				WithdrawalRequests: []*enginev1.ExecutionLayerWithdrawalRequest{
					withdrawalRequest,
				},
			}}
			RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, s state.BeaconState, b interfaces.SignedBeaconBlock) (state.BeaconState, error) {
				bod, ok := b.Block().(interfaces.ROBlockBodyElectra)
				require.Equal(t, true, ok)
				e, err := bod.Execution()
				require.NoError(t, err)
				exe, ok := e.(interfaces.ExecutionDataElectra)
				require.NoError(t, err)
				return electra.ProcessExecutionLayerWithdrawRequests(ctx, s, exe.WithdrawalRequests())
			})
		})
	}
}
