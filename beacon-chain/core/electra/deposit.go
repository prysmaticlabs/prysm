package electra

import (
	"context"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func ProcessDepositReceipts(ctx context.Context, beaconState state.BeaconState, receipts []*enginev1.DepositReceipt) (state.BeaconState, error) {
	if beaconState.Version() < version.Electra {
		return beaconState, nil
	}
	//	# Set deposit receipt start index
	//	if state.deposit_receipts_start_index == UNSET_DEPOSIT_RECEIPTS_START_INDEX:
	//	state.deposit_receipts_start_index = deposit_receipt.index
	//
	//	apply_deposit(
	//		state=state,
	//		pubkey=deposit_receipt.pubkey,
	//		withdrawal_credentials=deposit_receipt.withdrawal_credentials,
	//		amount=deposit_receipt.amount,
	//		signature=deposit_receipt.signature,
	//)
	deposits := make([]*eth.Deposit, len(receipts))
	for i, receipt := range receipts {
		deposits[i] = &eth.Deposit{
			Data: &eth.Deposit_Data{
				PublicKey:             bytesutil.SafeCopyBytes(receipt.Pubkey),
				WithdrawalCredentials: bytesutil.SafeCopyBytes(receipt.WithdrawalCredentials),
				Signature:             bytesutil.SafeCopyBytes(receipt.Signature),
			},
		}
	}
	return altair.ProcessDeposits(ctx, beaconState, deposits)
}
