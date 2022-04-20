package state_native

import (
	"fmt"

	nativetypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// SetCurrentSyncCommittee for the beacon state.
func (b *BeaconState) SetCurrentSyncCommittee(val *ethpb.SyncCommittee) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return fmt.Errorf("SetCurrentSyncCommittee is not supported for %s", version.String(b.version))
	}

	b.currentSyncCommittee = val
	b.markFieldAsDirty(nativetypes.CurrentSyncCommittee)
	return nil
}

// SetNextSyncCommittee for the beacon state.
func (b *BeaconState) SetNextSyncCommittee(val *ethpb.SyncCommittee) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return fmt.Errorf("SetNextSyncCommittee is not supported for %s", version.String(b.version))
	}

	b.nextSyncCommittee = val
	b.markFieldAsDirty(nativetypes.NextSyncCommittee)
	return nil
}
