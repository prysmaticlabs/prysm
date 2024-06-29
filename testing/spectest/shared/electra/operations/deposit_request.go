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

func RunDepositRequestsTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "electra", "operations/deposit_request/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			depositRequestFile, err := util.BazelFileBytes(folderPath, "deposit_request.ssz_snappy")
			require.NoError(t, err)
			depositRequestSSZ, err := snappy.Decode(nil /* dst */, depositRequestFile)
			require.NoError(t, err, "Failed to decompress")
			depositRequest := &enginev1.DepositRequest{}
			require.NoError(t, depositRequest.UnmarshalSSZ(depositRequestSSZ), "failed to unmarshal")
			requests := []*enginev1.DepositRequest{depositRequest}
			body := &ethpb.BeaconBlockBodyElectra{ExecutionPayload: &enginev1.ExecutionPayloadElectra{DepositRequests: requests}}
			processDepositRequestsFunc := func(ctx context.Context, s state.BeaconState, signedBlock interfaces.ReadOnlySignedBeaconBlock) (state.BeaconState, error) {
				e, err := signedBlock.Block().Body().Execution()
				require.NoError(t, err, "Failed to get execution")
				ee, ok := e.(interfaces.ExecutionDataElectra)
				require.Equal(t, true, ok, "Invalid execution payload")
				return electra.ProcessDepositRequests(ctx, s, ee.DepositRequests())
			}
			RunBlockOperationTest(t, folderPath, body, processDepositRequestsFunc)
		})
	}
}
