package state_native

import (
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	_ "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val interfaces.ExecutionDataHeader) error {
	headerCopy, err := blocks.CopyExecutionDataHeader(val)
	if err != nil {
		return err
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version < version.Bellatrix {
		return errNotSupported("SetLatestExecutionPayloadHeader", b.version)
	}

	b.latestExecutionPayloadHeader = headerCopy
	b.markFieldAsDirty(nativetypes.LatestExecutionPayloadHeader)
	return nil
}
