package state_native

import (
	"github.com/pkg/errors"
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	_ "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val interfaces.ExecutionData) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 || b.version == version.Altair {
		return errNotSupported("SetLatestExecutionPayloadHeader", b.version)
	}
	if (b.version == version.Bellatrix) {
		header, ok := val.Proto().(*enginev1.ExecutionPayloadHeader)
		if !ok {
			return errors.New("value must be an execution payload header")
		}
		b.latestExecutionPayloadHeader = header
		b.markFieldAsDirty(nativetypes.LatestExecutionPayloadHeader)
	}
	if (b.version == version.EIP4844) {
		header, ok := val.Proto().(*enginev1.ExecutionPayloadHeader4844)
		if !ok {
			return errors.New("value must be an execution payload header")
		}
		b.latestExecutionPayloadHeader = header
		b.markFieldAsDirty(nativetypes.LatestExecutionPayloadHeader)
	}

	return nil
}
