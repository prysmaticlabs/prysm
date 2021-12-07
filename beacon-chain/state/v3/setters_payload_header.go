package v3

import ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val *ethpb.ExecutionPayloadHeader) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.latestExecutionPayloadHeader = val
	b.markFieldAsDirty(latestExecutionPayloadHeader)
	return nil
}
