package blockchain

import (
	"context"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/time/slots"
	"testing"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestService_VerifyWeakSubjectivityRoot(t *testing.T) {
	beaconDB := testDB.SetupDB(t)

	b := util.NewBeaconBlock()
	b.Block.Slot = 1792480
	blockEpoch := slots.ToEpoch(b.Block.Slot)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(b)))
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	tests := []struct {
		wsVerified     bool
		disabled       bool
		wantErr        error
		checkpt        *ethpb.Checkpoint
		finalizedEpoch types.Epoch
		name           string
	}{
		{
			name:     "nil root and epoch",
			disabled: true,
		},
		{
			name:           "not yet to verify, ws epoch higher than finalized epoch",
			checkpt:        &ethpb.Checkpoint{Root: bytesutil.PadTo([]byte{'a'}, 32), Epoch: blockEpoch},
			finalizedEpoch: blockEpoch - 1,
		},
		{
			name:           "can't find the block in DB",
			checkpt:        &ethpb.Checkpoint{Root: bytesutil.PadTo([]byte{'a'}, 32), Epoch: 1},
			finalizedEpoch: blockEpoch + 1,
			wantErr:        errWSBlockNotFound,
		},
		{
			name:           "can't find the block corresponds to ws epoch in DB",
			checkpt:        &ethpb.Checkpoint{Root: r[:], Epoch: blockEpoch - 2}, // Root belongs in epoch 1.
			finalizedEpoch: blockEpoch - 1,
			wantErr:        errWSBlockNotFoundInEpoch,
		},
		{
			name:           "can verify and pass",
			checkpt:        &ethpb.Checkpoint{Root: r[:], Epoch: blockEpoch},
			finalizedEpoch: blockEpoch + 1,
		},
		{
			name:           "equal epoch",
			checkpt:        &ethpb.Checkpoint{Root: r[:], Epoch: blockEpoch},
			finalizedEpoch: blockEpoch,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wv, err := NewWeakSubjectivityVerifier(tt.checkpt, beaconDB)
			require.Equal(t, !tt.disabled, wv.enabled)
			require.NoError(t, err)
			s := &Service{
				cfg:              &config{BeaconDB: beaconDB, WeakSubjectivityCheckpt: tt.checkpt},
				finalizedCheckpt: &ethpb.Checkpoint{Epoch: tt.finalizedEpoch},
				wsVerifier:       wv,
			}
			err = s.wsVerifier.VerifyWeakSubjectivity(context.Background(), s.finalizedCheckpt.Epoch)
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, true, errors.Is(err, tt.wantErr))
			}
		})
	}
}
