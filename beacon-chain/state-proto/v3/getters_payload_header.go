package v3

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (*ethpb.ExecutionPayloadHeader, error) {
	if !b.hasInnerState() {
		return nil, nil
	}
	if b.state.LatestExecutionPayloadHeader == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.latestExecutionPayloadHeader(), nil
}

// latestExecutionPayloadHeader of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestExecutionPayloadHeader() *ethpb.ExecutionPayloadHeader {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyExecutionPayloadHeader(b.state.LatestExecutionPayloadHeader)
}
