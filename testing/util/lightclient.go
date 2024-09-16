package util

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpbv1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

type TestLightClient struct {
	T              *testing.T
	Ctx            context.Context
	State          state.BeaconState
	Block          interfaces.ReadOnlySignedBeaconBlock
	AttestedState  state.BeaconState
	AttestedHeader *ethpb.BeaconBlockHeader
	FinalizedBlock interfaces.ReadOnlySignedBeaconBlock
}

func NewTestLightClient(t *testing.T) *TestLightClient {
	return &TestLightClient{T: t}
}

func (l *TestLightClient) SetupTestCapella(blinded bool) *TestLightClient {
	ctx := context.Background()

	slot := primitives.Slot(params.BeaconConfig().CapellaForkEpoch * primitives.Epoch(params.BeaconConfig().SlotsPerEpoch)).Add(1)

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

	var signedBlock interfaces.SignedBeaconBlock
	if blinded {
		block := NewBlindedBeaconBlockCapella()
		block.Block.Slot = slot
		block.Block.ParentRoot = parentRoot[:]

		for i := uint64(0); i < params.BeaconConfig().MinSyncCommitteeParticipants; i++ {
			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
		}

		signedBlock, err = blocks.NewSignedBeaconBlock(block)
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
	} else {
		block := NewBeaconBlockCapella()
		block.Block.Slot = slot
		block.Block.ParentRoot = parentRoot[:]

		for i := uint64(0); i < params.BeaconConfig().MinSyncCommitteeParticipants; i++ {
			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
		}

		signedBlock, err = blocks.NewSignedBeaconBlock(block)
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
	}

	l.State = state
	l.AttestedState = attestedState
	l.AttestedHeader = attestedHeader
	l.Block = signedBlock
	l.Ctx = ctx

	return l
}

func (l *TestLightClient) SetupTestAltair() *TestLightClient {
	ctx := context.Background()

	slot := primitives.Slot(params.BeaconConfig().AltairForkEpoch * primitives.Epoch(params.BeaconConfig().SlotsPerEpoch)).Add(1)

	attestedState, err := NewBeaconStateAltair()
	require.NoError(l.T, err)
	err = attestedState.SetSlot(slot)
	require.NoError(l.T, err)

	parent := NewBeaconBlockAltair()
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

	state, err := NewBeaconStateAltair()
	require.NoError(l.T, err)
	err = state.SetSlot(slot)
	require.NoError(l.T, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(l.T, err)

	block := NewBeaconBlockAltair()
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

func (l *TestLightClient) SetupTestDeneb(blinded bool) *TestLightClient {
	ctx := context.Background()

	slot := primitives.Slot(params.BeaconConfig().DenebForkEpoch * primitives.Epoch(params.BeaconConfig().SlotsPerEpoch)).Add(1)

	attestedState, err := NewBeaconStateDeneb()
	require.NoError(l.T, err)
	err = attestedState.SetSlot(slot)
	require.NoError(l.T, err)

	parent := NewBeaconBlockDeneb()
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

	state, err := NewBeaconStateDeneb()
	require.NoError(l.T, err)
	err = state.SetSlot(slot)
	require.NoError(l.T, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(l.T, err)

	var signedBlock interfaces.SignedBeaconBlock
	if blinded {
		block := NewBlindedBeaconBlockDeneb()
		block.Message.Slot = slot
		block.Message.ParentRoot = parentRoot[:]

		for i := uint64(0); i < params.BeaconConfig().MinSyncCommitteeParticipants; i++ {
			block.Message.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
		}

		signedBlock, err = blocks.NewSignedBeaconBlock(block)
		require.NoError(l.T, err)

		h, err := signedBlock.Header()
		require.NoError(l.T, err)

		err = state.SetLatestBlockHeader(h.Header)
		require.NoError(l.T, err)
		stateRoot, err := state.HashTreeRoot(ctx)
		require.NoError(l.T, err)

		// get a new signed block so the root is updated with the new state root
		block.Message.StateRoot = stateRoot[:]
		signedBlock, err = blocks.NewSignedBeaconBlock(block)
		require.NoError(l.T, err)
	} else {
		block := NewBeaconBlockDeneb()
		block.Block.Slot = slot
		block.Block.ParentRoot = parentRoot[:]

		for i := uint64(0); i < params.BeaconConfig().MinSyncCommitteeParticipants; i++ {
			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
		}

		signedBlock, err = blocks.NewSignedBeaconBlock(block)
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
	}

	l.State = state
	l.AttestedState = attestedState
	l.AttestedHeader = attestedHeader
	l.Block = signedBlock
	l.Ctx = ctx

	return l
}

func (l *TestLightClient) CheckAttestedHeader(container *ethpbv2.LightClientHeaderContainer) {
	updateAttestedHeaderBeacon, err := container.GetBeacon()
	require.NoError(l.T, err)
	require.Equal(l.T, l.AttestedHeader.Slot, updateAttestedHeaderBeacon.Slot, "Attested header slot is not equal")
	require.Equal(l.T, l.AttestedHeader.ProposerIndex, updateAttestedHeaderBeacon.ProposerIndex, "Attested header proposer index is not equal")
	require.DeepSSZEqual(l.T, l.AttestedHeader.ParentRoot, updateAttestedHeaderBeacon.ParentRoot, "Attested header parent root is not equal")
	require.DeepSSZEqual(l.T, l.AttestedHeader.BodyRoot, updateAttestedHeaderBeacon.BodyRoot, "Attested header body root is not equal")

	attestedStateRoot, err := l.AttestedState.HashTreeRoot(l.Ctx)
	require.NoError(l.T, err)
	require.DeepSSZEqual(l.T, attestedStateRoot[:], updateAttestedHeaderBeacon.StateRoot, "Attested header state root is not equal")
}

func (l *TestLightClient) CheckSyncAggregate(sa *ethpbv1.SyncAggregate) {
	syncAggregate, err := l.Block.Block().Body().SyncAggregate()
	require.NoError(l.T, err)
	require.DeepSSZEqual(l.T, syncAggregate.SyncCommitteeBits, sa.SyncCommitteeBits, "SyncAggregate bits is not equal")
	require.DeepSSZEqual(l.T, syncAggregate.SyncCommitteeSignature, sa.SyncCommitteeSignature, "SyncAggregate signature is not equal")
}
