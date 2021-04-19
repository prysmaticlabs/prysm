package stateV0

import (
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// PreviousEpochAttestations of the beacon state.
func (b *BeaconState) LatestExecutionPayloadHeader() (*pbp2p.ExecutionPayloadHeader, error) {
	if !b.hasInnerState() {
		return nil, nil
	}
	if b.state.LatestExecutionPayloadHeader == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.latestExecutionPayloadHeader(), nil
}

// previousEpochAttestations of the  beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestExecutionPayloadHeader() *pbp2p.ExecutionPayloadHeader {
	if !b.hasInnerState() {
		return nil
	}

	return CopyExecutionPayloadHeader(b.state.LatestExecutionPayloadHeader)
}

// CopyExecutionPayloadHeader copies the provided execution payload object.
func CopyExecutionPayloadHeader(payload *pbp2p.ExecutionPayloadHeader) *pbp2p.ExecutionPayloadHeader {
	if payload == nil {
		return nil
	}
	return &pbp2p.ExecutionPayloadHeader{
		BlockHash:        bytesutil.SafeCopyBytes(payload.BlockHash),
		ParentHash:       bytesutil.SafeCopyBytes(payload.ParentHash),
		Coinbase:         bytesutil.SafeCopyBytes(payload.Coinbase),
		StateRoot:        bytesutil.SafeCopyBytes(payload.StateRoot),
		Number:           payload.Number,
		GasLimit:         payload.GasLimit,
		GasUsed:          payload.GasUsed,
		Timestamp:        payload.Timestamp,
		ReceiptRoot:      payload.ReceiptRoot,
		LogsBloom:        bytesutil.SafeCopyBytes(payload.LogsBloom),
		TransactionsRoot: bytesutil.SafeCopyBytes(payload.TransactionsRoot),
	}
}

func executionPayloadRoot(payload *pbp2p.ExecutionPayloadHeader) ([32]byte, error) {
	return payload.HashTreeRoot()
}
