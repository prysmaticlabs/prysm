// +build minimal

package field_params_test

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/assert"
)

func TestFieldParametersValues(t *testing.T) {
	params.UseMinimalConfig()
	assert.Equal(t, uint64(params.BeaconConfig().SlotsPerHistoricalRoot), uint64(fieldparams.BlockRootsLength))
	assert.Equal(t, uint64(params.BeaconConfig().SlotsPerHistoricalRoot), uint64(fieldparams.StateRootsLength))
	assert.Equal(t, params.BeaconConfig().HistoricalRootsLimit, uint64(fieldparams.HistoricalRootsLength))
	assert.Equal(t, uint64(params.BeaconConfig().EpochsPerHistoricalVector), uint64(fieldparams.RandaoMixesLength))
	assert.Equal(t, params.BeaconConfig().ValidatorRegistryLimit, uint64(fieldparams.ValidatorRegistryLimit))
	assert.Equal(t, uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))), uint64(fieldparams.Eth1DataVotesLength))
	assert.Equal(t, uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)), uint64(fieldparams.PreviousEpochAttestationsLength))
	assert.Equal(t, uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)), uint64(fieldparams.CurrentEpochAttestationsLength))
	assert.Equal(t, uint64(params.BeaconConfig().EpochsPerSlashingsVector), uint64(fieldparams.SlashingsLength))
	assert.Equal(t, params.BeaconConfig().SyncCommitteeSize, uint64(fieldparams.SyncCommitteeLength))
}
