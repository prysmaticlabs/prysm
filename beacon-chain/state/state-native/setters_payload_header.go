package state_native

import (
	"github.com/pkg/errors"
	nativetypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	_ "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// SetLatestExecutionPayloadHeader for the beacon state.
func (b *BeaconState) SetLatestExecutionPayloadHeader(val interfaces.ExecutionData) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 || b.version == version.Altair {
		return errNotSupported("SetLatestExecutionPayloadHeader", b.version)
	}
	header, ok := val.Proto().(*enginev1.ExecutionPayloadHeader)
	if !ok {
		return errors.New("value must be an execution payload header")
	}
	b.latestExecutionPayloadHeader = header
	b.markFieldAsDirty(nativetypes.LatestExecutionPayloadHeader)
	return nil
}
