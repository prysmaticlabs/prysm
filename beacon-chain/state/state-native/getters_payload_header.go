package state_native

import (
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (interfaces.WrappedExecutionPayloadHeader, error) {
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
func (b *BeaconState) latestExecutionPayloadHeaderVal() interfaces.WrappedExecutionPayloadHeader {
	if b.latestExecutionPayloadHeader == nil {
		return nil
	}

	switch payloadHeader := b.latestExecutionPayloadHeader.Proto().(type) {
	case *enginev1.ExecutionPayloadHeader:
		copy := ethpb.CopyExecutionPayloadHeader(payloadHeader)
		if copy == nil {
			return nil
		}
		copiedHeader, err := blocks.WrappedExecutionPayloadHeader(copy)
		if err != nil {
			return nil
		}
		return copiedHeader
	case *enginev1.ExecutionPayloadHeader4844:
		copy := ethpb.CopyExecutionPayloadHeader4844(payloadHeader)
		if copy == nil {
			return nil
		}
		copiedHeader, err := blocks.WrappedExecutionPayloadHeader(copy)
		if err != nil {
			return nil
		}
		return copiedHeader
	default:
		return nil // TODO: Should panic or return an error or something
	}
}
