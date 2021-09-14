package proposals

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/slasher/detection/proposals/iface"
	testDetect "github.com/prysmaticlabs/prysm/slasher/detection/testing"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

var _ iface.ProposalsDetector = (*ProposeDetector)(nil)

func TestProposalsDetector_DetectSlashingsForBlockHeaders(t *testing.T) {
	type testStruct struct {
		name        string
		blk         *ethpb.SignedBeaconBlockHeader
		incomingBlk *ethpb.SignedBeaconBlockHeader
		slashing    *ethpb.ProposerSlashing
	}
	s0, err := core.StartSlot(0)
	require.NoError(t, err)
	blk1slot0, err := testDetect.SignedBlockHeader(s0, 0)
	require.NoError(t, err)
	blk2slot0, err := testDetect.SignedBlockHeader(s0, 0)
	require.NoError(t, err)
	blk1slot1, err := testDetect.SignedBlockHeader(s0+1, 0)
	require.NoError(t, err)
	s1, err := core.StartSlot(1)
	require.NoError(t, err)
	blk1epoch1, err := testDetect.SignedBlockHeader(s1, 0)
	require.NoError(t, err)
	tests := []testStruct{
		{
			name:        "same block sig dont slash",
			blk:         blk1slot0,
			incomingBlk: blk1slot0,
			slashing:    nil,
		},
		{
			name:        "block from different epoch dont slash",
			blk:         blk1slot0,
			incomingBlk: blk1epoch1,
			slashing:    nil,
		},
		{
			name:        "different sig from different slot dont slash",
			blk:         blk1slot0,
			incomingBlk: blk1slot1,
			slashing:    nil,
		},
		{
			name:        "different sig from same slot slash",
			blk:         blk1slot0,
			incomingBlk: blk2slot0,
			slashing:    &ethpb.ProposerSlashing{Header_1: blk2slot0, Header_2: blk1slot0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testDB.SetupSlasherDB(t, false)
			ctx := context.Background()

			sd := &ProposeDetector{
				slasherDB: db,
			}

			require.NoError(t, sd.slasherDB.SaveBlockHeader(ctx, tt.blk))

			res, err := sd.DetectDoublePropose(ctx, tt.incomingBlk)
			require.NoError(t, err)
			assert.DeepEqual(t, tt.slashing, res)
		})
	}
}
