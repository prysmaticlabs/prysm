package state_native

import (
	"github.com/pkg/errors"
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	_ "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val interfaces.WrappedExecutionPayloadHeader) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 || b.version == version.Altair {
		return errNotSupported("SetLatestExecutionPayloadHeader", b.version)
	}

	b.latestExecutionPayloadHeader = val
	b.markFieldAsDirty(nativetypes.LatestExecutionPayloadHeader)

	return errors.New("value must be an execution payload header")
}
