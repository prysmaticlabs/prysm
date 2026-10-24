package state_native

import (
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
)

// TargetCheckpoint is the target of the current height, its root is empty until the next slot fills it in.
func (b *BeaconState) TargetCheckpoint() (*ethpb.HeightCheckpoint, error) {
	if b.version < version.EIP8453 {
		return nil, errNotSupported("TargetCheckpoint", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.targetCheckpoint, nil
}

// JustifiedCheckpointEIP8453 is the latest target that reached a quorum of target votes.
func (b *BeaconState) JustifiedCheckpointEIP8453() (*ethpb.HeightCheckpoint, error) {
	if b.version < version.EIP8453 {
		return nil, errNotSupported("JustifiedCheckpointEIP8453", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.justifiedCheckpointEIP8453, nil
}

// FinalizedCheckpointEIP8453 is the latest finalized justification.
func (b *BeaconState) FinalizedCheckpointEIP8453() (*ethpb.HeightCheckpoint, error) {
	if b.version < version.EIP8453 {
		return nil, errNotSupported("FinalizedCheckpointEIP8453", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.finalizedCheckpointEIP8453, nil
}

// TargetSlot is the slot of the block that brought the chain into the current height.
func (b *BeaconState) TargetSlot() (primitives.Slot, error) {
	if b.version < version.EIP8453 {
		return 0, errNotSupported("TargetSlot", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.targetSlot, nil
}

// FinalizedSlot is the slot of the finalized block.
func (b *BeaconState) FinalizedSlot() (primitives.Slot, error) {
	if b.version < version.EIP8453 {
		return 0, errNotSupported("FinalizedSlot", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.finalizedSlot, nil
}

// TargetParticipationEIP8453 marks the target votes at the current height, one byte per validator.
func (b *BeaconState) TargetParticipationEIP8453() ([]byte, error) {
	if b.version < version.EIP8453 {
		return nil, errNotSupported("TargetParticipationEIP8453", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.targetParticipationEIP8453, nil
}

// ProgressParticipation marks the target votes and timeouts at the current height, one byte per validator.
func (b *BeaconState) ProgressParticipation() ([]byte, error) {
	if b.version < version.EIP8453 {
		return nil, errNotSupported("ProgressParticipation", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.progressParticipation, nil
}

// RoundParticipation holds the participation flags of the current round, one byte per validator.
func (b *BeaconState) RoundParticipation() ([]byte, error) {
	if b.version < version.EIP8453 {
		return nil, errNotSupported("RoundParticipation", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.roundParticipation, nil
}

// PendingFinalities are the justifications not yet finalized, in increasing height order.
func (b *BeaconState) PendingFinalities() ([]*ethpb.PendingFinalityEIP8453, error) {
	if b.version < version.EIP8453 {
		return nil, errNotSupported("PendingFinalities", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.pendingFinalities, nil
}

// HeadVoteParticipation marks the head vote seats that voted for the block of their slot, previous epoch first.
func (b *BeaconState) HeadVoteParticipation() ([][]byte, error) {
	if b.version < version.EIP8453 {
		return nil, errNotSupported("HeadVoteParticipation", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.headVoteParticipation, nil
}
