package blockchain

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/store"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
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
		wantErr        error
		checkpt        *ethpb.Checkpoint
		finalizedEpoch types.Epoch
		name           string
	}{
		{
			name: "nil root and epoch",
		},
		{
			name:           "already verified",
			checkpt:        &ethpb.Checkpoint{Epoch: 2},
			finalizedEpoch: 2,
			wsVerified:     true,
		},
		{
			name:           "not yet to verify, ws epoch higher than finalized epoch",
			checkpt:        &ethpb.Checkpoint{Epoch: 2},
			finalizedEpoch: 1,
		},
		{
			name:           "can't find the block in DB",
			checkpt:        &ethpb.Checkpoint{Root: bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength), Epoch: 1},
			finalizedEpoch: 3,
			wantErr:        errWSBlockNotFound,
		},
		{
			name:           "can't find the block corresponds to ws epoch in DB",
			checkpt:        &ethpb.Checkpoint{Root: r[:], Epoch: 2}, // Root belongs in epoch 1.
			finalizedEpoch: 3,
			wantErr:        errWSBlockNotFoundInEpoch,
		},
		{
			name:           "can verify and pass",
			checkpt:        &ethpb.Checkpoint{Root: r[:], Epoch: 1},
			finalizedEpoch: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wv, err := NewWeakSubjectivityVerifier(tt.checkpt, beaconDB)
			require.NoError(t, err)
			s := &Service{
				cfg:        &config{BeaconDB: beaconDB, WeakSubjectivityCheckpt: tt.checkpt},
				store:      &store.Store{},
				wsVerifier: wv,
			}
			s.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Epoch: tt.finalizedEpoch})
			err = s.wsVerifier.VerifyWeakSubjectivity(context.Background(), s.store.FinalizedCheckpt().Epoch)
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, true, errors.Is(err, tt.wantErr))
			}
		})
	}
}
