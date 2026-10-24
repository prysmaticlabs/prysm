package state_native

import (
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
)

// SetTargetCheckpoint sets the target of the current height.
func (b *BeaconState) SetTargetCheckpoint(val *ethpb.HeightCheckpoint) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetTargetCheckpoint", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.targetCheckpoint = val
	b.markFieldAsDirty(types.TargetCheckpoint)
	return nil
}

// SetJustifiedCheckpointEIP8453 sets the latest justified target.
func (b *BeaconState) SetJustifiedCheckpointEIP8453(val *ethpb.HeightCheckpoint) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetJustifiedCheckpointEIP8453", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.justifiedCheckpointEIP8453 = val
	b.markFieldAsDirty(types.JustifiedCheckpointEIP8453)
	return nil
}

// SetFinalizedCheckpointEIP8453 sets the latest finalized justification.
func (b *BeaconState) SetFinalizedCheckpointEIP8453(val *ethpb.HeightCheckpoint) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetFinalizedCheckpointEIP8453", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.finalizedCheckpointEIP8453 = val
	b.markFieldAsDirty(types.FinalizedCheckpointEIP8453)
	return nil
}

// SetTargetSlot sets the slot of the block that brought the chain into the current height.
func (b *BeaconState) SetTargetSlot(val primitives.Slot) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetTargetSlot", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.targetSlot = val
	b.markFieldAsDirty(types.TargetSlot)
	return nil
}

// SetFinalizedSlot sets the slot of the finalized block.
func (b *BeaconState) SetFinalizedSlot(val primitives.Slot) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetFinalizedSlot", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.finalizedSlot = val
	b.markFieldAsDirty(types.FinalizedSlot)
	return nil
}

// SetTargetParticipationEIP8453 replaces the target votes at the current height.
func (b *BeaconState) SetTargetParticipationEIP8453(val []byte) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetTargetParticipationEIP8453", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.targetParticipationEIP8453 = val
	b.markFieldAsDirty(types.TargetParticipationEIP8453)
	return nil
}

// SetProgressParticipation replaces the target votes and timeouts at the current height.
func (b *BeaconState) SetProgressParticipation(val []byte) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetProgressParticipation", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.progressParticipation = val
	b.markFieldAsDirty(types.ProgressParticipation)
	return nil
}

// SetRoundParticipation replaces the participation flags of the current round.
func (b *BeaconState) SetRoundParticipation(val []byte) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetRoundParticipation", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.roundParticipation = val
	b.markFieldAsDirty(types.RoundParticipation)
	return nil
}

// SetPendingFinalities replaces the justifications not yet finalized.
func (b *BeaconState) SetPendingFinalities(val []*ethpb.PendingFinalityEIP8453) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetPendingFinalities", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.pendingFinalities = val
	b.markFieldAsDirty(types.PendingFinalities)
	return nil
}

// SetHeadVoteParticipation replaces the head vote seats recorded per slot.
func (b *BeaconState) SetHeadVoteParticipation(val [][]byte) error {
	if b.version < version.EIP8453 {
		return errNotSupported("SetHeadVoteParticipation", b.version)
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.headVoteParticipation = val
	b.markFieldAsDirty(types.HeadVoteParticipation)
	return nil
}
