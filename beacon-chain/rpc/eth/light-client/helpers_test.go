package lightclient

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
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

func (l *testlc) setupTest_SyncCommitteeBits_Equals_Zero() *testlc {
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

func (l *testlc) setupTest_SyncCommitteeBits_Equals_MoreThanTwoThirds() *testlc {
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
		if block.Block.Body.SyncAggregate.SyncCommitteeBits.Count()*3 >= params.BeaconConfig().MinSyncCommitteeParticipants*2 {
			break
		}
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

func (l *testlc) setupTest_SyncCommitteeBits_Equals_Many() *testlc {
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

func (l *testlc) checkAttestedHeader(update *ethpbv2.LightClientUpdate) {
	require.Equal(l.t, l.attestedHeader.Slot, update.AttestedHeader.Slot, "Attested header slot is not equal")
	require.Equal(l.t, l.attestedHeader.ProposerIndex, update.AttestedHeader.ProposerIndex, "Attested header proposer index is not equal")
	require.DeepSSZEqual(l.t, l.attestedHeader.ParentRoot, update.AttestedHeader.ParentRoot, "Attested header parent root is not equal")
	require.DeepSSZEqual(l.t, l.attestedHeader.BodyRoot, update.AttestedHeader.BodyRoot, "Attested header body root is not equal")

	attestedStateRoot, err := l.attestedState.HashTreeRoot(l.ctx)
	require.NoError(l.t, err)
	require.DeepSSZEqual(l.t, attestedStateRoot[:], update.AttestedHeader.StateRoot, "Attested header state root is not equal")
}

func (l *testlc) checkSyncAggregate(update *ethpbv2.LightClientUpdate) {
	syncAggregate, err := l.block.Block().Body().SyncAggregate()
	require.NoError(l.t, err)
	require.DeepSSZEqual(l.t, syncAggregate.SyncCommitteeBits, update.SyncAggregate.SyncCommitteeBits, "SyncAggregate bits is not equal")
	require.DeepSSZEqual(l.t, syncAggregate.SyncCommitteeSignature, update.SyncAggregate.SyncCommitteeSignature, "SyncAggregate signature is not equal")
}

func TestIsBetterUpdate_newHasSupermajority_notEquals_oldHasSupermajority(t *testing.T) {
	l_old := newTestLc(t).setupTest_SyncCommitteeBits_Equals_Zero()
	l_new := newTestLc(t).setupTest_SyncCommitteeBits_Equals_MoreThanTwoThirds()

	oldUpdate, err := blockchain.NewLightClientOptimisticUpdateFromBeaconState(l_old.ctx, l_old.state, l_old.block, l_old.attestedState)
	require.NoError(t, err)
	require.NotNil(t, oldUpdate, "old update is nil")
	newUpdate, err := blockchain.NewLightClientOptimisticUpdateFromBeaconState(l_new.ctx, l_new.state, l_new.block, l_new.attestedState)
	require.NoError(t, err)
	require.NotNil(t, newUpdate, "new update is nil")

	l_old.checkAttestedHeader(oldUpdate)
	l_old.checkSyncAggregate(oldUpdate)
	l_new.checkAttestedHeader(newUpdate)
	l_new.checkSyncAggregate(newUpdate)

	require.Equal(t, true, IsBetterUpdate(newUpdate, oldUpdate))
}
