package spectest

import (
	"context"
	"path"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func runProposerSlashingTest(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "operations/proposer_slashing/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			proposerSlashingFile, err := testutil.BazelFileBytes(folderPath, "proposer_slashing.ssz")
			require.NoError(t, err)
			proposerSlashing := &ethpb.ProposerSlashing{}
			require.NoError(t, proposerSlashing.UnmarshalSSZ(proposerSlashingFile), "Failed to unmarshal")

			body := &ethpb.BeaconBlockBody{ProposerSlashings: []*ethpb.ProposerSlashing{proposerSlashing}}
			testutil.RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, s iface.BeaconState, b *ethpb.SignedBeaconBlock) (iface.BeaconState, error) {
				return blocks.ProcessProposerSlashings(ctx, s, b.Block.Body.ProposerSlashings, v.SlashValidator)
			})
		})
	}
}
