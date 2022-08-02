package v3

import (
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (interfaces.ExecutionData, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.LatestExecutionPayloadHeader == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return wrapper.WrappedExecutionPayloadHeader(b.latestExecutionPayloadHeader())
}

// latestExecutionPayloadHeader of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestExecutionPayloadHeader() *enginev1.ExecutionPayloadHeader {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyExecutionPayloadHeader(b.state.LatestExecutionPayloadHeader)
}
