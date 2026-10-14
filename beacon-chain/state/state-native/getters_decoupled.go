package state_native

import (
	"github.com/OffchainLabs/go-bitfield"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
)

// Height is the height being voted on.
func (b *BeaconState) Height() (primitives.Height, error) {
	if b.version < version.Decoupled {
		return 0, errNotSupported("Height", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.height, nil
}

// Entry is the block that opened the current height.
func (b *BeaconState) Entry() (*ethpb.CheckpointDecoupled, error) {
	if b.version < version.Decoupled {
		return nil, errNotSupported("Entry", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.entry, nil
}

// TargetParticipation is the target votes at the current height, one bit per validator.
func (b *BeaconState) TargetParticipation() (bitfield.Bitlist, error) {
	if b.version < version.Decoupled {
		return nil, errNotSupported("TargetParticipation", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.targetParticipation, nil
}

// Progress is the target votes and timeouts at the current height, one bit per validator.
func (b *BeaconState) Progress() (bitfield.Bitlist, error) {
	if b.version < version.Decoupled {
		return nil, errNotSupported("Progress", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.progress, nil
}

// JustifiedHeight is the height of the latest justified entry.
func (b *BeaconState) JustifiedHeight() (primitives.Height, error) {
	if b.version < version.Decoupled {
		return 0, errNotSupported("JustifiedHeight", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.justifiedHeight, nil
}

// Justified is the latest justified entry.
func (b *BeaconState) Justified() (*ethpb.CheckpointDecoupled, error) {
	if b.version < version.Decoupled {
		return nil, errNotSupported("Justified", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.justified, nil
}

// FinalizedHeight is the height of the latest finalized entry.
func (b *BeaconState) FinalizedHeight() (primitives.Height, error) {
	if b.version < version.Decoupled {
		return 0, errNotSupported("FinalizedHeight", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.finalizedHeight, nil
}

// Finalized is the latest finalized entry.
func (b *BeaconState) Finalized() (*ethpb.CheckpointDecoupled, error) {
	if b.version < version.Decoupled {
		return nil, errNotSupported("Finalized", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.finalized, nil
}

// PendingFinality is the justified entries awaiting finality with their votes.
func (b *BeaconState) PendingFinality() ([]*ethpb.PendingFinality, error) {
	if b.version < version.Decoupled {
		return nil, errNotSupported("PendingFinality", b.version)
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.pendingFinality, nil
}
