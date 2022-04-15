package state_native

import (
	nativetypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val *ethpb.ExecutionPayloadHeader) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.latestExecutionPayloadHeader = val
	b.markFieldAsDirty(nativetypes.LatestExecutionPayloadHeader)
	return nil
}
