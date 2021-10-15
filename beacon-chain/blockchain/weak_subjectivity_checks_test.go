package blockchain

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestService_VerifyWeakSubjectivityRoot(t *testing.T) {
	beaconDB := testDB.SetupDB(t)

	b := util.NewBeaconBlock()
	b.Block.Slot = 32
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(b)))
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	tests := []struct {
		wsVerified     bool
		wantErr        bool
		checkpt        *ethpb.Checkpoint
		finalizedEpoch types.Epoch
		errString      string
		name           string
	}{
		{
			name:    "nil root and epoch",
			wantErr: false,
		},
		{
			name:           "already verified",
			checkpt:        &ethpb.Checkpoint{Epoch: 2},
			finalizedEpoch: 2,
			wsVerified:     true,
			wantErr:        false,
		},
		{
			name:           "not yet to verify, ws epoch higher than finalized epoch",
			checkpt:        &ethpb.Checkpoint{Epoch: 2},
			finalizedEpoch: 1,
			wantErr:        false,
		},
		{
			name:           "can't find the block in DB",
			checkpt:        &ethpb.Checkpoint{Root: bytesutil.PadTo([]byte{'a'}, 32), Epoch: 1},
			finalizedEpoch: 3,
			wantErr:        true,
			errString:      "node does not have root in DB",
		},
		{
			name:           "can't find the block corresponds to ws epoch in DB",
			checkpt:        &ethpb.Checkpoint{Root: r[:], Epoch: 2}, // Root belongs in epoch 1.
			finalizedEpoch: 3,
			wantErr:        true,
			errString:      "node does not have root in db corresponding to epoch",
		},
		{
			name:           "can verify and pass",
			checkpt:        &ethpb.Checkpoint{Root: r[:], Epoch: 1},
			finalizedEpoch: 3,
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				cfg:              &config{BeaconDB: beaconDB, WeakSubjectivityCheckpt: tt.checkpt},
				wsVerified:       tt.wsVerified,
				finalizedCheckpt: &ethpb.Checkpoint{Epoch: tt.finalizedEpoch},
			}
			if err := s.VerifyWeakSubjectivityRoot(context.Background()); (err != nil) != tt.wantErr {
				require.ErrorContains(t, tt.errString, err)
			}
		})
	}
}
