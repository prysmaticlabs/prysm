package blockchain

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_VerifyWeakSubjectivityRoot(t *testing.T) {
	beaconDB := testDB.SetupDB(t)

	b := testutil.NewBeaconBlock()
	b.Block.Slot = 32
	require.NoError(t, beaconDB.SaveBlock(context.Background(), b))
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	tests := []struct {
		wsVerified     bool
		wantErr        bool
		wsRoot         [32]byte
		wsEpoch        types.Epoch
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
			wsEpoch:        2,
			finalizedEpoch: 2,
			wsVerified:     true,
			wantErr:        false,
		},
		{
			name:           "not yet to verify, ws epoch higher than finalized epoch",
			wsEpoch:        2,
			finalizedEpoch: 1,
			wantErr:        false,
		},
		{
			name:           "can't find the block in DB",
			wsEpoch:        1,
			wsRoot:         [32]byte{'a'},
			finalizedEpoch: 3,
			wantErr:        true,
			errString:      "node does not have root in DB",
		},
		{
			name:           "can't find the block corresponds to ws epoch in DB",
			wsEpoch:        2,
			wsRoot:         r, // Root belongs in epoch 1.
			finalizedEpoch: 3,
			wantErr:        true,
			errString:      "node does not have root in db corresponding to epoch",
		},
		{
			name:           "can verify and pass",
			wsEpoch:        1,
			wsRoot:         r,
			finalizedEpoch: 3,
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				cfg:              &Config{BeaconDB: beaconDB, WspBlockRoot: tt.wsRoot[:], WspEpoch: tt.wsEpoch},
				wsVerified:       tt.wsVerified,
				finalizedCheckpt: &ethpb.Checkpoint{Epoch: tt.finalizedEpoch},
			}
			if err := s.VerifyWeakSubjectivityRoot(context.Background()); (err != nil) != tt.wantErr {
				require.ErrorContains(t, tt.errString, err)
			}
		})
	}
}
