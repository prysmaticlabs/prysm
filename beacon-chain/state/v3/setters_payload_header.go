package v3

import enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val *enginev1.ExecutionPayloadHeader) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.LatestExecutionPayloadHeader = val
	b.markFieldAsDirty(latestExecutionPayloadHeader)
	return nil
}
