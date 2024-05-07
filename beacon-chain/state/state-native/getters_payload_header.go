package state_native

import (
	"fmt"
	"math/big"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// LatestExecutionPayloadHeader of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (interfaces.ExecutionData, error) {
	if b.version < version.Bellatrix {
		return nil, errNotSupported("LatestExecutionPayloadHeader", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	switch b.version {
	case version.Bellatrix:
		return blocks.WrappedExecutionPayloadHeader(b.latestExecutionPayloadHeaderVal())
	case version.Capella:
		return blocks.WrappedExecutionPayloadHeaderCapella(b.latestExecutionPayloadHeaderCapellaVal(), big.NewInt(0))
	case version.Deneb:
		return blocks.WrappedExecutionPayloadHeaderDeneb(b.latestExecutionPayloadHeaderDenebVal(), big.NewInt(0))
	case version.Electra:
		return blocks.WrappedExecutionPayloadHeaderElectra(b.latestExecutionPayloadHeaderElectraVal(), big.NewInt(0))
	default:
		return nil, fmt.Errorf("unsupported version (%s) for latest execution payload header", version.String(b.version))
	}
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

func (b *BeaconState) latestExecutionPayloadHeaderDenebVal() *enginev1.ExecutionPayloadHeaderDeneb {
	return ethpb.CopyExecutionPayloadHeaderDeneb(b.latestExecutionPayloadHeaderDeneb)
}

func (b *BeaconState) latestExecutionPayloadHeaderElectraVal() *enginev1.ExecutionPayloadHeaderElectra {
	return ethpb.CopyExecutionPayloadHeaderElectra(b.latestExecutionPayloadHeaderElectra)
}
