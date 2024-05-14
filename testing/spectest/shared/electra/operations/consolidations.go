package operations

import (
	"context"
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/electra"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func RunConsolidationTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "electra", "operations/consolidation/pyspec_tests")
	require.NotEqual(t, 0, len(testFolders), "missing tests for consolidation operation in folder")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			consolidationFile, err := util.BazelFileBytes(folderPath, "consolidation.ssz_snappy")
			require.NoError(t, err)
			consolidationSSZ, err := snappy.Decode(nil /* dst */, consolidationFile)
			require.NoError(t, err, "Failed to decompress")
			consolidation := &ethpb.SignedConsolidation{}
			require.NoError(t, consolidation.UnmarshalSSZ(consolidationSSZ), "Failed to unmarshal")

			body := &ethpb.BeaconBlockBodyElectra{Consolidations: []*ethpb.SignedConsolidation{consolidation}}
			processConsolidationFunc := func(ctx context.Context, s state.BeaconState, b interfaces.SignedBeaconBlock) (state.BeaconState, error) {
				body, ok := b.Block().Body().(interfaces.ROBlockBodyElectra)
				if !ok {
					t.Error("block body is not electra")
				}
				cs := body.Consolidations()
				if len(cs) == 0 {
					t.Error("no consolidations to test")
				}
				return s, electra.ProcessConsolidations(ctx, s, cs)
			}
			RunBlockOperationTest(t, folderPath, body, processConsolidationFunc)
		})
	}
}
