package v1

import "github.com/prysmaticlabs/prysm/v3/config/params"

func (b *BeaconState) ProportionalSlashingMultiplier() (uint64, error) {
	return params.BeaconConfig().ProportionalSlashingMultiplier, nil
}

func (b *BeaconState) InactivityPenaltyQuotient() (uint64, error) {
	return params.BeaconConfig().InactivityPenaltyQuotient, nil
}
