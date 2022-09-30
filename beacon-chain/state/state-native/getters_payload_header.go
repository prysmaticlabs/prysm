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
	if b.version == version.Phase0 || b.version == version.Altair {
		return nil, errNotSupported("LatestExecutionPayloadHeader", b.version)
	}

	if b.latestExecutionPayloadHeader == nil {
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

	payloadHeader, err := b.latestExecutionPayloadHeader.Proto()
	if err != nil {
		return nil, err
	}

	switch h := payloadHeader.(type) {
	case *enginev1.ExecutionPayloadHeader:
		headerCpy := ethpb.CopyExecutionPayloadHeader(h)
		copiedHeader, err := blocks.NewExecutionDataHeader(headerCpy)
		if err != nil {
			return nil, err
		}
		return copiedHeader, nil
	case *enginev1.ExecutionPayloadHeader4844:
		headerCpy := ethpb.CopyExecutionPayloadHeader4844(h)
		copiedHeader, err := blocks.NewExecutionDataHeader(headerCpy)
		if err != nil {
			return nil, err
		}
		return copiedHeader, nil
	default:
		return nil, fmt.Errorf("invalid payload header in beacon state: %T", payloadHeader)
	}
}
