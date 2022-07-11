package wrapper

import (
	"bytes"

	"github.com/pkg/errors"
	fastssz "github.com/prysmaticlabs/fastssz"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"google.golang.org/protobuf/proto"
)

// bellatrixSignedBeaconBlock is a convenience wrapper around a Bellatrix blinded beacon block
// object. This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across prysm without issues.
type executionPayload struct {
	p *enginev1.ExecutionPayload
}

// WrappedBellatrixSignedBeaconBlock is a constructor which wraps a protobuf Bellatrix block with the block wrapper.
func WrappedExecutionPayload(p *enginev1.ExecutionPayload) (interfaces.ExecutionData, error) {
	w := executionPayload{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// IsNil checks if the underlying beacon block is nil.
func (e executionPayload) IsNil() bool {
	return e.p == nil
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (e executionPayload) MarshalSSZ() ([]byte, error) {
	return e.p.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block's ssz
// form to the provided byte buffer.
func (e executionPayload) MarshalSSZTo(dst []byte) ([]byte, error) {
	return e.p.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized signed block
func (e executionPayload) SizeSSZ() int {
	return e.p.SizeSSZ()
}

// UnmarshalSSZ unmarshals the signed beacon block from its relevant ssz
// form.
func (e executionPayload) UnmarshalSSZ(buf []byte) error {
	return e.p.UnmarshalSSZ(buf)
}

func (e executionPayload) HashTreeRoot() ([32]byte, error) {
	return e.p.HashTreeRoot()
}

func (e executionPayload) HashTreeRootWith(hh *fastssz.Hasher) error {
	return e.p.HashTreeRootWith(hh)
}

// Proto returns the block in its underlying protobuf interface.
func (e executionPayload) Proto() proto.Message {
	return e.p
}

func (e executionPayload) ParentHash() []byte {
	return e.p.ParentHash
}

func (e executionPayload) FeeRecipient() []byte {
	return e.p.FeeRecipient
}

func (e executionPayload) StateRoot() []byte {
	return e.p.StateRoot
}

func (e executionPayload) ReceiptsRoot() []byte {
	return e.p.ReceiptsRoot
}

func (e executionPayload) LogsBloom() []byte {
	return e.p.LogsBloom
}

func (e executionPayload) PrevRandao() []byte {
	return e.p.PrevRandao
}
func (e executionPayload) BlockNumber() uint64 {
	return e.p.BlockNumber
}

func (e executionPayload) GasLimit() uint64 {
	return e.p.GasLimit
}

func (e executionPayload) GasUsed() uint64 {
	return e.p.GasUsed
}

func (e executionPayload) Timestamp() uint64 {
	return e.p.Timestamp
}

func (e executionPayload) ExtraData() []byte {
	return e.p.ExtraData
}

func (e executionPayload) BaseFeePerGas() []byte {
	return e.p.BaseFeePerGas
}

func (e executionPayload) BlockHash() []byte {
	return e.p.BlockHash
}

func (e executionPayload) Transactions() ([][]byte, error) {
	return e.p.Transactions, nil
}

func (executionPayload) TransactionsRoot() ([]byte, error) {
	return nil, ErrUnsupportedField
}

type executionPayloadHeader struct {
	p *enginev1.ExecutionPayloadHeader
}

func WrappedExecutionPayloadHeader(p *enginev1.ExecutionPayloadHeader) (interfaces.ExecutionData, error) {
	w := executionPayloadHeader{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// IsNil checks if the underlying beacon block is nil.
func (e executionPayloadHeader) IsNil() bool {
	return e.p == nil
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (e executionPayloadHeader) MarshalSSZ() ([]byte, error) {
	return e.p.MarshalSSZ()
}

// MarshalSSZTo marshals the signed beacon block's ssz
// form to the provided byte buffer.
func (e executionPayloadHeader) MarshalSSZTo(dst []byte) ([]byte, error) {
	return e.p.MarshalSSZTo(dst)
}

// SizeSSZ returns the size of the serialized signed block
func (e executionPayloadHeader) SizeSSZ() int {
	return e.p.SizeSSZ()
}

// UnmarshalSSZ unmarshals the signed beacon block from its relevant ssz
// form.
func (e executionPayloadHeader) UnmarshalSSZ(buf []byte) error {
	return e.p.UnmarshalSSZ(buf)
}

func (e executionPayloadHeader) HashTreeRoot() ([32]byte, error) {
	return e.p.HashTreeRoot()
}

func (e executionPayloadHeader) HashTreeRootWith(hh *fastssz.Hasher) error {
	return e.p.HashTreeRootWith(hh)
}

// Proto returns the block in its underlying protobuf interface.
func (e executionPayloadHeader) Proto() proto.Message {
	return e.p
}
func (e executionPayloadHeader) ParentHash() []byte {
	return e.p.ParentHash
}

func (e executionPayloadHeader) FeeRecipient() []byte {
	return e.p.FeeRecipient
}

func (e executionPayloadHeader) StateRoot() []byte {
	return e.p.StateRoot
}

func (e executionPayloadHeader) ReceiptsRoot() []byte {
	return e.p.ReceiptsRoot
}

func (e executionPayloadHeader) LogsBloom() []byte {
	return e.p.LogsBloom
}

func (e executionPayloadHeader) PrevRandao() []byte {
	return e.p.PrevRandao
}
func (e executionPayloadHeader) BlockNumber() uint64 {
	return e.p.BlockNumber
}

func (e executionPayloadHeader) GasLimit() uint64 {
	return e.p.GasLimit
}

func (e executionPayloadHeader) GasUsed() uint64 {
	return e.p.GasUsed
}

func (e executionPayloadHeader) Timestamp() uint64 {
	return e.p.Timestamp
}

func (e executionPayloadHeader) ExtraData() []byte {
	return e.p.ExtraData
}

func (e executionPayloadHeader) BaseFeePerGas() []byte {
	return e.p.BaseFeePerGas
}

func (e executionPayloadHeader) BlockHash() []byte {
	return e.p.BlockHash
}

func (executionPayloadHeader) Transactions() ([][]byte, error) {
	return nil, ErrUnsupportedField
}

func (e executionPayloadHeader) TransactionsRoot() ([]byte, error) {
	return e.p.TransactionsRoot, nil
}

func IsEmptyExecutionData(data interfaces.ExecutionData) (bool, error) {
	if !bytes.Equal(data.ParentHash(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.FeeRecipient(), make([]byte, fieldparams.FeeRecipientLength)) {
		return false, nil
	}
	if !bytes.Equal(data.StateRoot(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.ReceiptsRoot(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.LogsBloom(), make([]byte, fieldparams.LogsBloomLength)) {
		return false, nil
	}
	if !bytes.Equal(data.PrevRandao(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.BaseFeePerGas(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}
	if !bytes.Equal(data.BlockHash(), make([]byte, fieldparams.RootLength)) {
		return false, nil
	}

	txs, err := data.Transactions()
	switch {
	case errors.Is(err, ErrUnsupportedField):
	case err != nil:
		return false, err
	default:
		if len(txs) != 0 {
			return false, nil
		}
	}

	txsRoot, err := data.TransactionsRoot()
	switch {
	case errors.Is(err, ErrUnsupportedField):
	case err != nil:
		return false, err
	default:
		if !bytes.Equal(txsRoot, make([]byte, fieldparams.RootLength)) {
			return false, nil
		}
	}
	if len(data.ExtraData()) != 0 {
		return false, nil
	}
	if data.BlockNumber() != 0 {
		return false, nil
	}
	if data.GasLimit() != 0 {
		return false, nil
	}
	if data.GasUsed() != 0 {
		return false, nil
	}
	if data.Timestamp() != 0 {
		return false, nil
	}
	return true, nil
}
