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

	proto := val.Proto()
	headerOld, okOld := proto.(*enginev1.ExecutionPayloadHeader)
	headerNew, okNew := proto.(*enginev1.ExecutionPayloadHeader4844)

	if okNew {
		b.latestExecutionPayloadHeader = headerNew
		b.markFieldAsDirty(nativetypes.LatestExecutionPayloadHeader)
		return nil
	}

	if okOld {
		b.latestExecutionPayloadHeader = headerOld
		b.markFieldAsDirty(nativetypes.LatestExecutionPayloadHeader)
		return nil
	}

	return errors.New("value must be an execution payload header")
}
