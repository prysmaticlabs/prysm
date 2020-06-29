package spectest

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"path"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runDepositTest(t *testing.T, config string) {
	if err := spectest.SetConfig(t, config); err != nil {
		t.Fatal(err)
	}

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "operations/deposit/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			depositFile, err := testutil.BazelFileBytes(folderPath, "deposit.ssz")
			if err != nil {
				t.Fatal(err)
			}
			deposit := &ethpb.Deposit{}
			if err := deposit.UnmarshalSSZ(depositFile); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			body := &ethpb.BeaconBlockBody{Deposits: []*ethpb.Deposit{deposit}}
			testutil.RunBlockOperationTest(t, folderPath, body, func(ctx context.Context, state *state.BeaconState, body *ethpb.BeaconBlockBody) (*state.BeaconState, error) {
				return blocks.ProcessDeposits(ctx, state, body.Deposits)
			})
		})
	}
}
