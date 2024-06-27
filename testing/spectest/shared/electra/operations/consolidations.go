package operations

import (
	"path"
	"testing"

	"github.com/golang/snappy"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func RunConsolidationTest(t *testing.T, config string) {
	t.Skip("Failing until spectests are updated to v1.5.0-alpha.3")
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
			consolidation := &enginev1.ConsolidationRequest{}
			require.NoError(t, consolidation.UnmarshalSSZ(consolidationSSZ), "Failed to unmarshal")

			t.Fatal("Implement me")
		})
	}
}
