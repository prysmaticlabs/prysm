package stateV0

import pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val *pbp2p.ExecutionPayloadHeader) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.LatestExecutionPayloadHeader = val
	b.markFieldAsDirty(latestExecutionPayloadHeader)
	return nil
}
