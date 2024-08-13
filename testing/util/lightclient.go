package util

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"testing"

	"context"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db/kv"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

type TestLightClient struct {
	T              *testing.T
	Ctx            context.Context
	State          state.BeaconState
	Block          interfaces.ReadOnlySignedBeaconBlock
	AttestedState  state.BeaconState
	AttestedHeader *ethpb.BeaconBlockHeader
}

func NewTestLightClient(t *testing.T) *TestLightClient {
	return &TestLightClient{T: t}
}

func (l *TestLightClient) SetupTest() *TestLightClient {
	ctx := context.Background()

	slot := primitives.Slot(params.BeaconConfig().AltairForkEpoch * primitives.Epoch(params.BeaconConfig().SlotsPerEpoch)).Add(1)

	attestedState, err := NewBeaconStateCapella()
	require.NoError(l.T, err)
	err = attestedState.SetSlot(slot)
	require.NoError(l.T, err)

	parent := NewBeaconBlockCapella()
	parent.Block.Slot = slot

	signedParent, err := blocks.NewSignedBeaconBlock(parent)
	require.NoError(l.T, err)

	parentHeader, err := signedParent.Header()
	require.NoError(l.T, err)
	attestedHeader := parentHeader.Header

	err = attestedState.SetLatestBlockHeader(attestedHeader)
	require.NoError(l.T, err)
	attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
	require.NoError(l.T, err)

	// get a new signed block so the root is updated with the new state root
	parent.Block.StateRoot = attestedStateRoot[:]
	signedParent, err = blocks.NewSignedBeaconBlock(parent)
	require.NoError(l.T, err)

	state, err := NewBeaconStateCapella()
	require.NoError(l.T, err)
	err = state.SetSlot(slot)
	require.NoError(l.T, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(l.T, err)

	block := NewBeaconBlockCapella()
	block.Block.Slot = slot
	block.Block.ParentRoot = parentRoot[:]

	for i := uint64(0); i < params.BeaconConfig().MinSyncCommitteeParticipants; i++ {
		block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
	}

	signedBlock, err := blocks.NewSignedBeaconBlock(block)
	require.NoError(l.T, err)

	h, err := signedBlock.Header()
	require.NoError(l.T, err)

	err = state.SetLatestBlockHeader(h.Header)
	require.NoError(l.T, err)
	stateRoot, err := state.HashTreeRoot(ctx)
	require.NoError(l.T, err)

	// get a new signed block so the root is updated with the new state root
	block.Block.StateRoot = stateRoot[:]
	signedBlock, err = blocks.NewSignedBeaconBlock(block)
	require.NoError(l.T, err)

	l.State = state
	l.AttestedState = attestedState
	l.AttestedHeader = attestedHeader
	l.Block = signedBlock
	l.Ctx = ctx

	return l
}

// SetupDB setupDB instantiates and returns a Store instance.
func SetupDB(t testing.TB) *kv.Store {
	db, err := kv.NewKVStore(context.Background(), t.TempDir())
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return db
}

func (l *TestLightClient) CheckAttestedHeader(update *ethpbv2.LightClientUpdate) {
	require.Equal(l.T, l.AttestedHeader.Slot, update.AttestedHeader.Slot, "Attested header slot is not equal")
	require.Equal(l.T, l.AttestedHeader.ProposerIndex, update.AttestedHeader.ProposerIndex, "Attested header proposer index is not equal")
	require.DeepSSZEqual(l.T, l.AttestedHeader.ParentRoot, update.AttestedHeader.ParentRoot, "Attested header parent root is not equal")
	require.DeepSSZEqual(l.T, l.AttestedHeader.BodyRoot, update.AttestedHeader.BodyRoot, "Attested header body root is not equal")

	attestedStateRoot, err := l.AttestedState.HashTreeRoot(l.Ctx)
	require.NoError(l.T, err)
	require.DeepSSZEqual(l.T, attestedStateRoot[:], update.AttestedHeader.StateRoot, "Attested header state root is not equal")
}

func (l *TestLightClient) CheckSyncAggregate(update *ethpbv2.LightClientUpdate) {
	syncAggregate, err := l.Block.Block().Body().SyncAggregate()
	require.NoError(l.T, err)
	require.DeepSSZEqual(l.T, syncAggregate.SyncCommitteeBits, update.SyncAggregate.SyncCommitteeBits, "SyncAggregate bits is not equal")
	require.DeepSSZEqual(l.T, syncAggregate.SyncCommitteeSignature, update.SyncAggregate.SyncCommitteeSignature, "SyncAggregate signature is not equal")
}
