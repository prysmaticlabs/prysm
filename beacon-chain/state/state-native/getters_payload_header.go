package state_native

import (
	types "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (types.ExecutionPayloadHeader, error) {
	if b.version == version.Phase0 || b.version == version.Altair {
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
func (b *BeaconState) latestExecutionPayloadHeaderVal() types.ExecutionPayloadHeader {

	switch payloadHeader := b.latestExecutionPayloadHeader.(type) {
	case *enginev1.ExecutionPayloadHeader:
		return ethpb.CopyExecutionPayloadHeader(payloadHeader)
	case *enginev1.ExecutionPayloadHeader4844:
		return ethpb.CopyExecutionPayloadHeader4844(payloadHeader)
	default:
		return nil // TODO: Should panic or return an error or something
	}
}
