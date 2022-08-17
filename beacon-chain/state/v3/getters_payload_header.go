package v3

import (
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (*enginev1.ExecutionPayloadHeader, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
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
func (b *BeaconState) latestExecutionPayloadHeader() *enginev1.ExecutionPayloadHeader {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyExecutionPayloadHeader(b.state.LatestExecutionPayloadHeader)
}
