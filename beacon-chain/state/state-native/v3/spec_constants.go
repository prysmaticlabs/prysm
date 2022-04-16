package v3

import "github.com/prysmaticlabs/prysm/config/params"

func (b *BeaconState) InactivityPenaltyQuotient() uint64 {
	return params.BeaconConfig().InactivityPenaltyQuotientBellatrix
}
