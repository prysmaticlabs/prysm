package types

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestStateFieldIndexes(t *testing.T) {
	assert.Equal(t, GenesisTime, 0)
	assert.Equal(t, GenesisValidatorRoot, 1)
	assert.Equal(t, Slot, 2)
	assert.Equal(t, Fork, 3)
	assert.Equal(t, LatestBlockHeader, 4)
	assert.Equal(t, BlockRoots, 5)
	assert.Equal(t, StateRoots, 6)
	assert.Equal(t, HistoricalRoots, 7)
	assert.Equal(t, Eth1Data, 8)
	assert.Equal(t, Eth1DataVotes, 9)
	assert.Equal(t, Eth1DepositIndex, 10)
	assert.Equal(t, Validators, 11)
	assert.Equal(t, Balances, 12)
	assert.Equal(t, RandaoMixes, 13)
	assert.Equal(t, Slashings, 14)
	assert.Equal(t, PreviousEpochAttestations, 15)
	assert.Equal(t, PreviousEpochParticipationBits, 15)
	assert.Equal(t, CurrentEpochAttestations, 16)
	assert.Equal(t, CurrentEpochParticipationBits, 16)
	assert.Equal(t, JustificationBits, 17)
	assert.Equal(t, PreviousJustifiedCheckpoint, 18)
	assert.Equal(t, CurrentJustifiedCheckpoint, 19)
	assert.Equal(t, FinalizedCheckpoint, 20)
	assert.Equal(t, InactivityScores, 21)
	assert.Equal(t, CurrentSyncCommittee, 22)
	assert.Equal(t, NextSyncCommittee, 23)
}
