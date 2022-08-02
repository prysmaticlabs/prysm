package state_native

import (
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (interfaces.ExecutionData, error) {
	if b.version < version.Bellatrix {
		return nil, errNotSupported("LatestExecutionPayloadHeader", b.version)
	}

	if b.latestExecutionPayloadHeader == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.version == version.Bellatrix {
		return wrapper.WrappedExecutionPayloadHeader(b.latestExecutionPayloadHeaderVal())
	}
	return wrapper.WrappedExecutionPayloadHeaderCapella(b.latestExecutionPayloadHeaderCapellaVal())
}

// latestExecutionPayloadHeaderVal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestExecutionPayloadHeaderVal() *enginev1.ExecutionPayloadHeader {
	return ethpb.CopyExecutionPayloadHeader(b.latestExecutionPayloadHeader)
}

// latestExecutionPayloadHeaderCapellaVal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestExecutionPayloadHeaderCapellaVal() *enginev1.ExecutionPayloadHeaderCapella {
	return ethpb.CopyExecutionPayloadHeaderCapella(b.latestExecutionPayloadHeaderCapella)
}
