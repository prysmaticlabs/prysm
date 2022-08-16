package v3

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
)

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val interfaces.ExecutionData) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	header, ok := val.Proto().(*enginev1.ExecutionPayloadHeader)
	if !ok {
		return errors.New("value must be an execution payload header")
	}
	b.state.LatestExecutionPayloadHeader = header
	b.markFieldAsDirty(latestExecutionPayloadHeader)
	return nil
}
