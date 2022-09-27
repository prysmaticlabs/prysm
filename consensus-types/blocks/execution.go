package blocks

import (
	"bytes"
	"errors"

	fastssz "github.com/prysmaticlabs/fastssz"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// executionPayload is a convenience wrapper around a beacon block body's execution payload data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayload struct {
	p interfaces.CommonExecutionPayloadData
}

// WrappedExecutionPayload is a constructor which wraps a protobuf execution payload into an interface.
func WrappedExecutionPayload(p interfaces.CommonExecutionPayloadData) (interfaces.WrappedExecutionPayload, error) {
	w := executionPayload{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// Only EIP-4844 blocks contain ExcessBlobs
func (e executionPayload) GetExcessBlobs() (uint64, error) {
	switch payload := e.p.(type) {
	case *enginev1.ExecutionPayload4844:
		return payload.GetExcessBlobs(), nil
	case *enginev1.ExecutionPayloadHeader4844:
		return payload.GetExcessBlobs(), nil
	}
	return 0, ErrUnsupportedGetter
}

// Transactions -- not present on Headers
func (e executionPayload) GetTransactions() ([][]byte, error) {
	switch payload := e.p.(type) {
	case *enginev1.ExecutionPayload:
		return payload.GetTransactions(), nil
	case *enginev1.ExecutionPayload4844:
		return payload.GetTransactions(), nil
	}

	// Headers don't have Transactions
	return nil, ErrUnsupportedGetter
}

// TransactionsRoot --
func (e executionPayload) GetTransactionsRoot() ([]byte, error) {
	// The headers already have it
	switch payload := e.p.(type) {
	case *enginev1.ExecutionPayloadHeader:
		return payload.GetTransactionsRoot(), nil
	case *enginev1.ExecutionPayloadHeader4844:
		return payload.GetTransactionsRoot(), nil
	}

	// Otherwise, we need to compute it from the transactions
	txs, err := e.GetTransactions()
	if err != nil {
		return nil, err
	}

	txRoot, err := ssz.TransactionsRoot(txs)
	if err != nil {
		return nil, err
	}
	return txRoot[:], nil
}

func (e executionPayload) Proto() proto.Message {
	return e.p
}

func (e executionPayload) ProtoReflect() protoreflect.Message {
	return e.p.ProtoReflect()
}

// IsNil checks if the underlying data is nil.
func (e executionPayload) IsNil() bool {
	return e.p == nil
}

// MarshalSSZ --
func (e executionPayload) MarshalSSZ() ([]byte, error) {
	return e.p.MarshalSSZ()
}

// MarshalSSZTo --
func (e executionPayload) MarshalSSZTo(dst []byte) ([]byte, error) {
	return e.p.MarshalSSZTo(dst)
}

// SizeSSZ --
func (e executionPayload) SizeSSZ() int {
	return e.p.SizeSSZ()
}

// UnmarshalSSZ --
func (e executionPayload) UnmarshalSSZ(buf []byte) error {
	return e.p.UnmarshalSSZ(buf)
}

// HashTreeRoot --
func (e executionPayload) HashTreeRoot() ([32]byte, error) {
	return e.p.HashTreeRoot()
}

// HashTreeRootWith --
func (e executionPayload) HashTreeRootWith(hh *fastssz.Hasher) error {
	return e.p.HashTreeRootWith(hh)
}

// ParentHash --
func (e executionPayload) GetParentHash() []byte {
	return e.p.GetParentHash()
}

// FeeRecipient --
func (e executionPayload) GetFeeRecipient() []byte {
	return e.p.GetFeeRecipient()
}

// StateRoot --
func (e executionPayload) GetStateRoot() []byte {
	return e.p.GetStateRoot()
}

// ReceiptsRoot --
func (e executionPayload) GetReceiptsRoot() []byte {
	return e.p.GetReceiptsRoot()
}

// LogsBloom --
func (e executionPayload) GetLogsBloom() []byte {
	return e.p.GetLogsBloom()
}

// PrevRandao --
func (e executionPayload) GetPrevRandao() []byte {
	return e.p.GetPrevRandao()
}

// BlockNumber --
func (e executionPayload) GetBlockNumber() uint64 {
	return e.p.GetBlockNumber()
}

// GasLimit --
func (e executionPayload) GetGasLimit() uint64 {
	return e.p.GetGasLimit()
}

// GasUsed --
func (e executionPayload) GetGasUsed() uint64 {
	return e.p.GetGasUsed()
}

// Timestamp --
func (e executionPayload) GetTimestamp() uint64 {
	return e.p.GetTimestamp()
}

// ExtraData --
func (e executionPayload) GetExtraData() []byte {
	return e.p.GetExtraData()
}

// BaseFeePerGas --
func (e executionPayload) GetBaseFeePerGas() []byte {
	return e.p.GetBaseFeePerGas()
}

// BlockHash --
func (e executionPayload) GetBlockHash() []byte {
	return e.p.GetBlockHash()
}

func (e executionPayload) ToHeader() (interfaces.WrappedExecutionPayloadHeader, error) {
	switch payload := e.p.(type) {
	case executionPayloadHeader:
		// Already wrapped
		return payload, nil
	case executionPayload:
		return payload.ToHeader()
	case *enginev1.ExecutionPayload:
		txs := payload.GetTransactions()
		txRoot, err := ssz.TransactionsRoot(txs)
		if err != nil {
			return nil, err
		}
		return WrappedExecutionPayloadHeader(&enginev1.ExecutionPayloadHeader{
			ParentHash:       bytesutil.SafeCopyBytes(payload.GetParentHash()),
			FeeRecipient:     bytesutil.SafeCopyBytes(payload.GetFeeRecipient()),
			StateRoot:        bytesutil.SafeCopyBytes(payload.GetStateRoot()),
			ReceiptsRoot:     bytesutil.SafeCopyBytes(payload.GetReceiptsRoot()),
			LogsBloom:        bytesutil.SafeCopyBytes(payload.GetLogsBloom()),
			PrevRandao:       bytesutil.SafeCopyBytes(payload.GetPrevRandao()),
			BlockNumber:      payload.GetBlockNumber(),
			GasLimit:         payload.GetGasLimit(),
			GasUsed:          payload.GetGasUsed(),
			Timestamp:        payload.GetTimestamp(),
			ExtraData:        bytesutil.SafeCopyBytes(payload.GetExtraData()),
			BaseFeePerGas:    bytesutil.SafeCopyBytes(payload.GetBaseFeePerGas()),
			BlockHash:        bytesutil.SafeCopyBytes(payload.GetBlockHash()),
			TransactionsRoot: txRoot[:],
		})
	case *enginev1.ExecutionPayload4844:
		txs := payload.GetTransactions()
		txRoot, err := ssz.TransactionsRoot(txs)
		if err != nil {
			return nil, err
		}
		return WrappedExecutionPayloadHeader(&enginev1.ExecutionPayloadHeader4844{
			ParentHash:       bytesutil.SafeCopyBytes(payload.GetParentHash()),
			FeeRecipient:     bytesutil.SafeCopyBytes(payload.GetFeeRecipient()),
			StateRoot:        bytesutil.SafeCopyBytes(payload.GetStateRoot()),
			ReceiptsRoot:     bytesutil.SafeCopyBytes(payload.GetReceiptsRoot()),
			LogsBloom:        bytesutil.SafeCopyBytes(payload.GetLogsBloom()),
			PrevRandao:       bytesutil.SafeCopyBytes(payload.GetPrevRandao()),
			BlockNumber:      payload.GetBlockNumber(),
			GasLimit:         payload.GetGasLimit(),
			GasUsed:          payload.GetGasUsed(),
			Timestamp:        payload.GetTimestamp(),
			ExtraData:        bytesutil.SafeCopyBytes(payload.GetExtraData()),
			BaseFeePerGas:    bytesutil.SafeCopyBytes(payload.GetBaseFeePerGas()),
			BlockHash:        bytesutil.SafeCopyBytes(payload.GetBlockHash()),
			ExcessBlobs:      payload.GetExcessBlobs(),
			TransactionsRoot: txRoot[:],
		})
	case *enginev1.ExecutionPayloadHeader:
		return WrappedExecutionPayloadHeader(payload)
	case *enginev1.ExecutionPayloadHeader4844:
		return WrappedExecutionPayloadHeader(payload)
	}

	return nil, errors.New("unknown execution payload")
}

// IsEmptyExecutionData checks if an execution data is empty underneath. If a single field has
// a non-zero value, this function will return false.
func IsEmptyExecutionData(data interfaces.CommonExecutionPayloadData) (bool, error) {
	if !bytes.Equal(data.GetParentHash(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.GetFeeRecipient(), make([]byte, fieldparams.FeeRecipientLength)) {
		return false, nil
	}
	if !bytes.Equal(data.GetStateRoot(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.GetReceiptsRoot(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.GetLogsBloom(), make([]byte, fieldparams.LogsBloomLength)) {
		return false, nil
	}
	if !bytes.Equal(data.GetPrevRandao(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.GetBaseFeePerGas(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.GetBlockHash(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if len(data.GetExtraData()) != 0 {
		return false, nil
	}
	if data.GetBlockNumber() != 0 {
		return false, nil
	}
	if data.GetGasLimit() != 0 {
		return false, nil
	}
	if data.GetGasUsed() != 0 {
		return false, nil
	}
	if data.GetTimestamp() != 0 {
		return false, nil
	}

	payload, err := WrappedExecutionPayload(data)
	if err != nil {
		return false, err
	}

	txs, err := payload.GetTransactions()
	switch {
	case errors.Is(err, ErrUnsupportedGetter):
	case err != nil:
		return false, err
	default:
		if len(txs) != 0 {
			return false, nil
		}
	}

	return true, nil
}
