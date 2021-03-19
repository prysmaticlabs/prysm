package slasher

import (
	"context"
	"testing"

	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestIsSlashableBlock(t *testing.T) {
	ctx := context.Background()
	beaconDB := dbtest.SetupDB(t)
	s := &Service{
		serviceCfg: &ServiceConfig{
			Database:              beaconDB,
			ProposerSlashingsFeed: new(event.Feed),
		},
		params:    DefaultParams(),
		blksQueue: newBlocksQueue(),
	}
	err := beaconDB.SaveBlockProposals(ctx, []*slashertypes.SignedBlockHeaderWrapper{
		createProposalWrapper(t, 2, 3, []byte{1}),
		createProposalWrapper(t, 3, 3, []byte{1}),
	})
	require.NoError(t, err)
	tests := []struct {
		name              string
		blockToCheck      *slashertypes.SignedBlockHeaderWrapper
		shouldBeSlashable bool
	}{
		{
			name:              "should not detect if same signing root",
			blockToCheck:      createProposalWrapper(t, 2, 3, []byte{1}),
			shouldBeSlashable: false,
		},
		{
			name:              "should not detect if different slot",
			blockToCheck:      createProposalWrapper(t, 1, 3, []byte{2}),
			shouldBeSlashable: false,
		},
		{
			name:              "should not detect if different validator index",
			blockToCheck:      createProposalWrapper(t, 2, 4, []byte{2}),
			shouldBeSlashable: false,
		},
		{
			name:              "detects differing signing root",
			blockToCheck:      createProposalWrapper(t, 2, 3, []byte{2}),
			shouldBeSlashable: true,
		},
		{
			name:              "should detect another slot",
			blockToCheck:      createProposalWrapper(t, 3, 3, []byte{2}),
			shouldBeSlashable: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposerSlashing, err := s.IsSlashableBlock(ctx, tt.blockToCheck.SignedBeaconBlockHeader)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldBeSlashable, proposerSlashing != nil)
		})
	}
}
