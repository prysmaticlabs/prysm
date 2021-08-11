package types

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestStateFieldIndexes(t *testing.T) {
	assert.Equal(t, GenesisTime, FieldIndex(0))
	assert.Equal(t, GenesisValidatorRoot, FieldIndex(1))
	assert.Equal(t, Slot, FieldIndex(2))
	assert.Equal(t, Fork, FieldIndex(3))
	assert.Equal(t, LatestBlockHeader, FieldIndex(4))
	assert.Equal(t, BlockRoots, FieldIndex(5))
	assert.Equal(t, StateRoots, FieldIndex(6))
	assert.Equal(t, HistoricalRoots, FieldIndex(7))
	assert.Equal(t, Eth1Data, FieldIndex(8))
	assert.Equal(t, Eth1DataVotes, FieldIndex(9))
	assert.Equal(t, Eth1DepositIndex, FieldIndex(10))
	assert.Equal(t, Validators, FieldIndex(11))
	assert.Equal(t, Balances, FieldIndex(12))
	assert.Equal(t, RandaoMixes, FieldIndex(13))
	assert.Equal(t, Slashings, FieldIndex(14))
	assert.Equal(t, PreviousEpochAttestations, FieldIndex(15))
	assert.Equal(t, PreviousEpochParticipationBits, FieldIndex(15))
	assert.Equal(t, CurrentEpochAttestations, FieldIndex(16))
	assert.Equal(t, CurrentEpochParticipationBits, FieldIndex(16))
	assert.Equal(t, JustificationBits, FieldIndex(17))
	assert.Equal(t, PreviousJustifiedCheckpoint, FieldIndex(18))
	assert.Equal(t, CurrentJustifiedCheckpoint, FieldIndex(19))
	assert.Equal(t, FinalizedCheckpoint, FieldIndex(20))
	assert.Equal(t, InactivityScores, FieldIndex(21))
	assert.Equal(t, CurrentSyncCommittee, FieldIndex(22))
	assert.Equal(t, NextSyncCommittee, FieldIndex(23))
}
