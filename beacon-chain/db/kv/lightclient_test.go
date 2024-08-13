package kv_test

import (
	"context"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db/kv"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

type testlc struct {
	t              *testing.T
	ctx            context.Context
	state          state.BeaconState
	block          interfaces.ReadOnlySignedBeaconBlock
	attestedState  state.BeaconState
	attestedHeader *ethpb.BeaconBlockHeader
}

func newTestLc(t *testing.T) *testlc {
	return &testlc{t: t}
}

func (l *testlc) setupTest() *testlc {
	ctx := context.Background()

	slot := primitives.Slot(params.BeaconConfig().AltairForkEpoch * primitives.Epoch(params.BeaconConfig().SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateCapella()
	require.NoError(l.t, err)
	err = attestedState.SetSlot(slot)
	require.NoError(l.t, err)

	parent := util.NewBeaconBlockCapella()
	parent.Block.Slot = slot

	signedParent, err := blocks.NewSignedBeaconBlock(parent)
	require.NoError(l.t, err)

	parentHeader, err := signedParent.Header()
	require.NoError(l.t, err)
	attestedHeader := parentHeader.Header

	err = attestedState.SetLatestBlockHeader(attestedHeader)
	require.NoError(l.t, err)
	attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
	require.NoError(l.t, err)

	// get a new signed block so the root is updated with the new state root
	parent.Block.StateRoot = attestedStateRoot[:]
	signedParent, err = blocks.NewSignedBeaconBlock(parent)
	require.NoError(l.t, err)

	state, err := util.NewBeaconStateCapella()
	require.NoError(l.t, err)
	err = state.SetSlot(slot)
	require.NoError(l.t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(l.t, err)

	block := util.NewBeaconBlockCapella()
	block.Block.Slot = slot
	block.Block.ParentRoot = parentRoot[:]

	for i := uint64(0); i < params.BeaconConfig().MinSyncCommitteeParticipants; i++ {
		block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
	}

	signedBlock, err := blocks.NewSignedBeaconBlock(block)
	require.NoError(l.t, err)

	h, err := signedBlock.Header()
	require.NoError(l.t, err)

	err = state.SetLatestBlockHeader(h.Header)
	require.NoError(l.t, err)
	stateRoot, err := state.HashTreeRoot(ctx)
	require.NoError(l.t, err)

	// get a new signed block so the root is updated with the new state root
	block.Block.StateRoot = stateRoot[:]
	signedBlock, err = blocks.NewSignedBeaconBlock(block)
	require.NoError(l.t, err)

	l.state = state
	l.attestedState = attestedState
	l.attestedHeader = attestedHeader
	l.block = signedBlock
	l.ctx = ctx

	return l
}

// setupDB instantiates and returns a Store instance.
func setupDB(t testing.TB) *kv.Store {
	db, err := kv.NewKVStore(context.Background(), t.TempDir())
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return db
}

func TestStore_LightclientUpdate_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)

	l := newTestLc(t).setupTest()

	update, err := blockchain.NewLightClientOptimisticUpdateFromBeaconState(l.ctx, l.state, l.block, l.attestedState)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	period := uint64(1)
	err = db.SaveLightClientUpdate(l.ctx, period, &ethpbv2.LightClientUpdateWithVersion{
		Version: 1,
		Data:    update,
	})
	require.NoError(t, err)

}
