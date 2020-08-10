package spectest

import (
	"context"
	"path"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func runDepositTest(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "operations/deposit/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			depositFile, err := testutil.BazelFileBytes(folderPath, "deposit.ssz")
			require.NoError(t, err)
			deposit := &ethpb.Deposit{}
			require.NoError(t, deposit.UnmarshalSSZ(depositFile), "Failed to unmarshal")

			body := &ethpb.BeaconBlockBody{Deposits: []*ethpb.Deposit{deposit}}
			testutil.RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, state *state.BeaconState, body *ethpb.BeaconBlockBody) (*state.BeaconState, error) {
				return blocks.ProcessDeposits(ctx, state, body.Deposits)
			})
		})
	}
}
