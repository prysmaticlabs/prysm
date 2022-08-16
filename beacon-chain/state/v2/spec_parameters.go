package v2

import "github.com/prysmaticlabs/prysm/v3/config/params"

func (b *BeaconState) ProportionalSlashingMultiplier() (uint64, error) {
	return params.BeaconConfig().ProportionalSlashingMultiplierAltair, nil
}

func (b *BeaconState) InactivityPenaltyQuotient() (uint64, error) {
	return params.BeaconConfig().InactivityPenaltyQuotientAltair, nil
}
