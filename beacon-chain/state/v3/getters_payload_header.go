package v3

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// hasInnerState detects if the internal reference to the state data structure
// is populated correctly. Returns false if nil.
func (b *BeaconState) hasInnerState() bool {
	return b != nil && b.state != nil
}

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

// previousEpochAttestations of the  beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestExecutionPayloadHeader() *ethpb.ExecutionPayloadHeader {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyExecutionPayloadHeader(b.state.LatestExecutionPayloadHeader)
}
