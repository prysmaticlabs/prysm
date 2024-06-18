package stateutil

import (
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

func PendingPartialWithdrawalsRoot(slice []*ethpb.PendingPartialWithdrawal) ([32]byte, error) {
	return SliceRoot(slice, fieldparams.PendingPartialWithdrawalsLimit)
}
