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
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func RunConsolidationTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "electra", "operations/consolidation_request/pyspec_tests")
	require.NotEqual(t, 0, len(testFolders), "missing tests for consolidation operation in folder")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			consolidationFile, err := util.BazelFileBytes(folderPath, "consolidation_request.ssz_snappy")
			require.NoError(t, err)
			consolidationSSZ, err := snappy.Decode(nil /* dst */, consolidationFile)
			require.NoError(t, err, "Failed to decompress")
			consolidation := &enginev1.ConsolidationRequest{}
			require.NoError(t, consolidation.UnmarshalSSZ(consolidationSSZ), "Failed to unmarshal")

			body := &eth.BeaconBlockBodyElectra{ExecutionPayload: &enginev1.ExecutionPayloadElectra{
				ConsolidationRequests: []*enginev1.ConsolidationRequest{consolidation},
			}}
			RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, s state.BeaconState, b interfaces.ReadOnlySignedBeaconBlock) (state.BeaconState, error) {
				ed, err := b.Block().Body().Execution()
				if err != nil {
					return nil, err
				}
				eed, ok := ed.(interfaces.ExecutionDataElectra)
				if !ok {
					t.Fatal("block does not have execution data for electra")
				}
				return s, electra.ProcessConsolidationRequests(ctx, s, eed.ConsolidationRequests())
			})
		})
	}
}
