package electra

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

func ProcessDepositReceipts(beaconState state.BeaconState, beaconBlock interfaces.ReadOnlyBeaconBlock) (state.BeaconState, error) {
	if beaconState.Version() < version.Electra {
		return beaconState, nil
	}
	payload, err := beaconBlock.Body().Execution()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload")
	}
	ede, ok := payload.(interfaces.ExecutionDataElectra)
	if !ok {
		return nil, errors.New("invalid electra execution payload")
	}
	for _, receipt := range ede.DepositReceipts() {
		beaconState, err = processDepositReceipt(beaconState, receipt)
		if err != nil {
			return nil, errors.Wrap(err, "could not apply deposit receipt")
		}
	}
	return beaconState, nil
}

// def process_deposit_receipt(state: BeaconState, deposit_receipt: DepositReceipt) -> None:
//
//	# Set deposit receipt start index
//	if state.deposit_receipts_start_index == UNSET_DEPOSIT_RECEIPTS_START_INDEX:
//	    state.deposit_receipts_start_index = deposit_receipt.index
//
//	apply_deposit(
//	    state=state,
//	    pubkey=deposit_receipt.pubkey,
//	    withdrawal_credentials=deposit_receipt.withdrawal_credentials,
//	    amount=deposit_receipt.amount,
//	    signature=deposit_receipt.signature,
//	)
func processDepositReceipt(beaconState state.BeaconState, receipt *enginev1.DepositReceipt) (state.BeaconState, error) {
	receiptsStartIndex, err := beaconState.DepositReceiptsStartIndex()
	if err != nil {
		return nil, errors.Wrap(err, "could not get deposit receipts start index")
	}
	if receiptsStartIndex == params.BeaconConfig().UnsetDepositReceiptsStartIndex {
		if err := beaconState.SetDepositReceiptsStartIndex(receipt.Index); err != nil {
			return nil, errors.Wrap(err, "could not set deposit receipts start index")
		}
	}
	return blocks.ApplyDeposit(beaconState, &eth.Deposit_Data{
		PublicKey:             bytesutil.SafeCopyBytes(receipt.Pubkey),
		WithdrawalCredentials: bytesutil.SafeCopyBytes(receipt.WithdrawalCredentials),
		Signature:             bytesutil.SafeCopyBytes(receipt.Signature),
	}, true) // individually verify signatures instead of batch verify
}
