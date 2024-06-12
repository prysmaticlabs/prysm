package operations

import (
	"context"
	"path"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/spectest/utils"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func RunDepositReceiptsTest(t *testing.T, config string) {
	require.NoError(t, utils.SetConfig(t, config))
	testFolders, testsFolderPath := utils.TestFolders(t, config, "electra", "operations/deposit_receipt/pyspec_tests")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			depositReceiptFile, err := util.BazelFileBytes(folderPath, "deposit_receipt.ssz_snappy")
			require.NoError(t, err)
			depositReceiptSSZ, err := snappy.Decode(nil /* dst */, depositReceiptFile)
			require.NoError(t, err, "Failed to decompress")
			depositReceipt := &enginev1.DepositReceipt{}
			require.NoError(t, depositReceipt.UnmarshalSSZ(depositReceiptSSZ), "failed to unmarshal")

			body := &ethpb.BeaconBlockBodyElectra{ExecutionPayload: &enginev1.ExecutionPayloadElectra{DepositReceipts: []*enginev1.DepositReceipt{depositReceipt}}}
			processDepositReceiptsFunc := func(ctx context.Context, s state.BeaconState, b interfaces.SignedBeaconBlock) (state.BeaconState, error) {
				return blocks.ProcessDepositReceipts(s, b.Block())
			}
			RunBlockOperationTest(t, folderPath, body, processDepositReceiptsFunc)
		})
	}
}
