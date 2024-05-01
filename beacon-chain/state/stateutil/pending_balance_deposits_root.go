package stateutil

import (
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

func PendingBalanceDepositsRoot(slice []*ethpb.PendingBalanceDeposit) ([32]byte, error) {
	return SliceRoot(slice, fieldparams.PendingBalanceDepositsLimit)
}
