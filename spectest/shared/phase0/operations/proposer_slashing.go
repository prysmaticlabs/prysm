package operations

import (
	"context"
	"path"
	"testing"

	"github.com/golang/snappy"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/spectest/utils"
)

func RunProposerSlashingTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "phase0", "operations/proposer_slashing/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			proposerSlashingFile, err := testutil.BazelFileBytes(folderPath, "proposer_slashing.ssz_snappy")
			require.NoError(t, err)
			proposerSlashingSSZ, err := snappy.Decode(nil /* dst */, proposerSlashingFile)
			require.NoError(t, err, "Failed to decompress")
			proposerSlashing := &ethpb.ProposerSlashing{}
			require.NoError(t, proposerSlashing.UnmarshalSSZ(proposerSlashingSSZ), "Failed to unmarshal")

			body := &ethpb.BeaconBlockBody{ProposerSlashings: []*ethpb.ProposerSlashing{proposerSlashing}}
			RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, s iface.BeaconState, b *ethpb.SignedBeaconBlock) (iface.BeaconState, error) {
				return blocks.ProcessProposerSlashings(ctx, s, b.Block.Body.ProposerSlashings, v.SlashValidator)
			})
		})
	}
}
