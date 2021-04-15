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

func runAttesterSlashingTest(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "operations/attester_slashing/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			attSlashingFile, err := testutil.BazelFileBytes(folderPath, "attester_slashing.ssz")
			require.NoError(t, err)
			attSlashing := &ethpb.AttesterSlashing{}
			require.NoError(t, attSlashing.UnmarshalSSZ(attSlashingFile), "Failed to unmarshal")

			body := &ethpb.BeaconBlockBody{AttesterSlashings: []*ethpb.AttesterSlashing{attSlashing}}
			testutil.RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, s iface.BeaconState, b *ethpb.SignedBeaconBlock) (iface.BeaconState, error) {
				return blocks.ProcessAttesterSlashings(ctx, s, b.Block.Body.AttesterSlashings, v.SlashValidator)
			})
		})
	}
}
