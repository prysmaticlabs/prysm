package blocks

import (
	fastssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// executionPayload is a convenience wrapper around a beacon block body's execution payload data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayloadHeader struct {
	p interfaces.ExecutionPayloadHeader
}

// WrappedExecutionPayload is a constructor which wraps a protobuf execution payload into an interface.
func WrappedExecutionPayloadHeader(p interfaces.ExecutionPayloadHeader) (interfaces.WrappedExecutionPayloadHeader, error) {
	w := executionPayloadHeader{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

func (e executionPayloadHeader) Proto() proto.Message {
	return e.p
}

func (e executionPayloadHeader) ProtoReflect() protoreflect.Message {
	return e.p.ProtoReflect()
}

// TransactionsRoot --
func (e executionPayloadHeader) GetTransactionsRoot() []byte {
	return e.p.GetTransactionsRoot()
}

// Only EIP-4844 blocks contain ExcessBlobs
func (e executionPayloadHeader) GetExcessBlobs() (uint64, error) {
	switch payload := e.p.(type) {
	case *enginev1.ExecutionPayloadHeader4844:
		return payload.GetExcessBlobs(), nil
	}
	return 0, ErrUnsupportedGetter
}

// IsNil checks if the underlying data is nil.
func (e executionPayloadHeader) IsNil() bool {
	return e.p == nil
}

// MarshalSSZ --
func (e executionPayloadHeader) MarshalSSZ() ([]byte, error) {
	return e.p.MarshalSSZ()
}

// MarshalSSZTo --
func (e executionPayloadHeader) MarshalSSZTo(dst []byte) ([]byte, error) {
	return e.p.MarshalSSZTo(dst)
}

// SizeSSZ --
func (e executionPayloadHeader) SizeSSZ() int {
	return e.p.SizeSSZ()
}

// UnmarshalSSZ --
func (e executionPayloadHeader) UnmarshalSSZ(buf []byte) error {
	return e.p.UnmarshalSSZ(buf)
}

// HashTreeRoot --
func (e executionPayloadHeader) HashTreeRoot() ([32]byte, error) {
	return e.p.HashTreeRoot()
}

// HashTreeRootWith --
func (e executionPayloadHeader) HashTreeRootWith(hh *fastssz.Hasher) error {
	return e.p.HashTreeRootWith(hh)
}

// ParentHash --
func (e executionPayloadHeader) GetParentHash() []byte {
	return e.p.GetParentHash()
}

// FeeRecipient --
func (e executionPayloadHeader) GetFeeRecipient() []byte {
	return e.p.GetFeeRecipient()
}

// StateRoot --
func (e executionPayloadHeader) GetStateRoot() []byte {
	return e.p.GetStateRoot()
}

// ReceiptsRoot --
func (e executionPayloadHeader) GetReceiptsRoot() []byte {
	return e.p.GetReceiptsRoot()
}

// LogsBloom --
func (e executionPayloadHeader) GetLogsBloom() []byte {
	return e.p.GetLogsBloom()
}

// PrevRandao --
func (e executionPayloadHeader) GetPrevRandao() []byte {
	return e.p.GetPrevRandao()
}

// BlockNumber --
func (e executionPayloadHeader) GetBlockNumber() uint64 {
	return e.p.GetBlockNumber()
}

// GasLimit --
func (e executionPayloadHeader) GetGasLimit() uint64 {
	return e.p.GetGasLimit()
}

// GasUsed --
func (e executionPayloadHeader) GetGasUsed() uint64 {
	return e.p.GetGasUsed()
}

// Timestamp --
func (e executionPayloadHeader) GetTimestamp() uint64 {
	return e.p.GetTimestamp()
}

// ExtraData --
func (e executionPayloadHeader) GetExtraData() []byte {
	return e.p.GetExtraData()
}

// BaseFeePerGas --
func (e executionPayloadHeader) GetBaseFeePerGas() []byte {
	return e.p.GetBaseFeePerGas()
}

// BlockHash --
func (e executionPayloadHeader) GetBlockHash() []byte {
	return e.p.GetBlockHash()
}
