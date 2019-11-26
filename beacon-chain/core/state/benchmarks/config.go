package benchmarks

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// ValidatorCount is for declaring how many validators the benchmarks will be
// performed with. Default is 65536 or 2M ETH staked.
var ValidatorCount = uint64(65536)

// AttestationsPerEpoch represents the requested amount attestations in an epoch.
// This affects the amount of attestations in a fully attested for block and the amount
// of attestations in the state per epoch, so a full 2 epochs should result in twice
// this amount of attestations in the state. Default is 128.
var AttestationsPerEpoch = uint64(128)

// GenesisFileName is the generated genesis beacon state file name.
var GenesisFileName = fmt.Sprintf("benchmark_files/bStateGenesis-%dAtts-%dVals.ssz", AttestationsPerEpoch, ValidatorCount)

// BState1EpochFileName is the generated beacon state after 1 skipped epoch file name.
var BState1EpochFileName = fmt.Sprintf("benchmark_files/bState1Epoch-%dAtts-%dVals.ssz", AttestationsPerEpoch, ValidatorCount)

// BState2EpochFileName is the generated beacon state after 2 full epochs file name.
var BState2EpochFileName = fmt.Sprintf("benchmark_files/bState2Epochs-%dAtts-%dVals.ssz", AttestationsPerEpoch, ValidatorCount)

// FullBlockFileName is the generated full block file name.
var FullBlockFileName = fmt.Sprintf("benchmark_files/fullBlock-%dAtts-%dVals.ssz", AttestationsPerEpoch, ValidatorCount)

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
