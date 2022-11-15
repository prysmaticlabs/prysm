package state_native

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (interfaces.ExecutionDataHeader, error) {
	if b.version < version.Bellatrix {
		return nil, errNotSupported("LatestExecutionPayloadHeader", b.version)
	}

	if b.latestExecutionPayloadHeader == nil || b.latestExecutionPayloadHeader.IsNil() {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.latestExecutionPayloadHeaderVal()
}

// latestExecutionPayloadHeaderVal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestExecutionPayloadHeaderVal() (interfaces.ExecutionDataHeader, error) {
	if b.latestExecutionPayloadHeader == nil {
		return nil, nil
	}

	headerProto := b.latestExecutionPayloadHeader.Proto()
	switch h := headerProto.(type) {
	case *enginev1.ExecutionPayloadHeader:
		headerCopy := ethpb.CopyExecutionPayloadHeader(h)
		return blocks.WrappedExecutionPayloadHeader(headerCopy)
	case *enginev1.ExecutionPayloadHeaderCapella:
		headerCopy := ethpb.CopyExecutionPayloadHeaderCapella(h)
		return blocks.WrappedExecutionPayloadHeaderCapella(headerCopy)
	default:
		return nil, fmt.Errorf("invalid payload header in beacon state: %T", headerProto)
	}
}
