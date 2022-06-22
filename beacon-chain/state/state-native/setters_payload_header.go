package state_native

import (
	nativetypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	_ "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val *enginev1.ExecutionPayloadHeader) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 || b.version == version.Altair {
		return errNotSupported("SetLatestExecutionPayloadHeader", b.version)
	}

	b.latestExecutionPayloadHeader = val
	b.markFieldAsDirty(nativetypes.LatestExecutionPayloadHeader)
	return nil
}
