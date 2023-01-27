package state

import (
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *State) LatestExecutionPayloadHeader() (interfaces.ExecutionData, error) {
	if b.version < version.Bellatrix {
		return nil, errNotSupported("LatestExecutionPayloadHeader", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.version == version.Bellatrix {
		return blocks.WrappedExecutionPayloadHeader(b.latestExecutionPayloadHeaderVal())
	}
	return blocks.WrappedExecutionPayloadHeaderCapella(b.latestExecutionPayloadHeaderCapellaVal())
}

// latestExecutionPayloadHeaderVal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *State) latestExecutionPayloadHeaderVal() *enginev1.ExecutionPayloadHeader {
	return ethpb.CopyExecutionPayloadHeader(b.latestExecutionPayloadHeader)
}

// latestExecutionPayloadHeaderCapellaVal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *State) latestExecutionPayloadHeaderCapellaVal() *enginev1.ExecutionPayloadHeaderCapella {
	return ethpb.CopyExecutionPayloadHeaderCapella(b.latestExecutionPayloadHeaderCapella)
}
