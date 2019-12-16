package benchmarks

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// FilePath prefixes the file path to the file names.
func FilePath(fileName string) string {
	return fmt.Sprintf("beacon-chain/core/state/benchmarks/%s", fileName)
}

// SetConfig changes the beacon config to match the requested amount of
// attestations set to AttestationsPerEpoch.
func SetConfig() {
	maxAtts := AttestationsPerEpoch
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	committeeSize := (ValidatorCount / slotsPerEpoch) / (maxAtts / slotsPerEpoch)
	c := params.BeaconConfig()
	c.PersistentCommitteePeriod = 0
	c.MinValidatorWithdrawabilityDelay = 0
	c.TargetCommitteeSize = committeeSize
	c.MaxAttestations = maxAtts
	params.OverrideBeaconConfig(c)
}
