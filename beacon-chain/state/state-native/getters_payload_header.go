package state_native

import (
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (*enginev1.ExecutionPayloadHeader, error) {
	if b.version != version.Bellatrix {
		return nil, errNotSupported("LatestExecutionPayloadHeader", b.version)
	}

	if b.latestExecutionPayloadHeader == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.latestExecutionPayloadHeaderVal(), nil
}

// latestExecutionPayloadHeaderVal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestExecutionPayloadHeaderVal() *enginev1.ExecutionPayloadHeader {
	return ethpb.CopyExecutionPayloadHeader(b.latestExecutionPayloadHeader)
}

// LatestExecutionPayloadHeaderCapella of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeaderCapella() (*enginev1.ExecutionPayloadHeaderCapella, error) {
	if b.version != version.Capella {
		return nil, errNotSupported("LatestExecutionPayloadHeaderCapella", b.version)
	}

	if b.latestExecutionPayloadHeaderCapella == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.latestExecutionPayloadHeaderCapellaVal(), nil
}

// latestExecutionPayloadHeaderCapellaVal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestExecutionPayloadHeaderCapellaVal() *enginev1.ExecutionPayloadHeaderCapella {
	return ethpb.CopyExecutionPayloadHeaderCapella(b.latestExecutionPayloadHeaderCapella)
}
