package blocks

import (
	"bytes"
	"errors"

	fastssz "github.com/prysmaticlabs/fastssz"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"google.golang.org/protobuf/proto"
)

// executionPayload is a convenience wrapper around a beacon block body's execution payload data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayload struct {
	p *enginev1.ExecutionPayload
}

// WrappedExecutionPayload is a constructor which wraps a protobuf execution payload into an interface.
func WrappedExecutionPayload(p *enginev1.ExecutionPayload) (interfaces.ExecutionData, error) {
	w := executionPayload{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
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

// Proto --
func (e executionPayload) Proto() proto.Message {
	return e.p
}

// ParentHash --
func (e executionPayload) ParentHash() []byte {
	return e.p.ParentHash
}

// FeeRecipient --
func (e executionPayload) FeeRecipient() []byte {
	return e.p.FeeRecipient
}

// StateRoot --
func (e executionPayload) StateRoot() []byte {
	return e.p.StateRoot
}

// ReceiptsRoot --
func (e executionPayload) ReceiptsRoot() []byte {
	return e.p.ReceiptsRoot
}

// LogsBloom --
func (e executionPayload) LogsBloom() []byte {
	return e.p.LogsBloom
}

// PrevRandao --
func (e executionPayload) PrevRandao() []byte {
	return e.p.PrevRandao
}

// BlockNumber --
func (e executionPayload) BlockNumber() uint64 {
	return e.p.BlockNumber
}

// GasLimit --
func (e executionPayload) GasLimit() uint64 {
	return e.p.GasLimit
}

// GasUsed --
func (e executionPayload) GasUsed() uint64 {
	return e.p.GasUsed
}

// Timestamp --
func (e executionPayload) Timestamp() uint64 {
	return e.p.Timestamp
}

// ExtraData --
func (e executionPayload) ExtraData() []byte {
	return e.p.ExtraData
}

// BaseFeePerGas --
func (e executionPayload) BaseFeePerGas() []byte {
	return e.p.BaseFeePerGas
}

// BlockHash --
func (e executionPayload) BlockHash() []byte {
	return e.p.BlockHash
}

// Transactions --
func (e executionPayload) Transactions() ([][]byte, error) {
	return e.p.Transactions, nil
}

// TransactionsRoot --
func (e executionPayload) TransactionsRoot() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

// Withdrawals --
func (e executionPayload) Withdrawals() ([]*enginev1.Withdrawal, error) {
	return nil, ErrUnsupportedGetter
}

// WithdrawalsRoot --
func (e executionPayload) WithdrawalsRoot() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

// ExcessiveDataGas --
func (e executionPayload) ExcessiveDataGas() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

// PbBellatrix --
func (e executionPayload) PbBellatrix() (*enginev1.ExecutionPayload, error) {
	return e.p, nil
}

// PbCapella --
func (executionPayload) PbCapella() (*enginev1.ExecutionPayloadCapella, error) {
	return nil, ErrUnsupportedGetter
}

// executionPayloadHeader is a convenience wrapper around a blinded beacon block body's execution header data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayloadHeader struct {
	p *enginev1.ExecutionPayloadHeader
}

// WrappedExecutionPayloadHeader is a constructor which wraps a protobuf execution header into an interface.
func WrappedExecutionPayloadHeader(p *enginev1.ExecutionPayloadHeader) (interfaces.ExecutionData, error) {
	w := executionPayloadHeader{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
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

// Proto --
func (e executionPayloadHeader) Proto() proto.Message {
	return e.p
}

// ParentHash --
func (e executionPayloadHeader) ParentHash() []byte {
	return e.p.ParentHash
}

// FeeRecipient --
func (e executionPayloadHeader) FeeRecipient() []byte {
	return e.p.FeeRecipient
}

// StateRoot --
func (e executionPayloadHeader) StateRoot() []byte {
	return e.p.StateRoot
}

// ReceiptsRoot --
func (e executionPayloadHeader) ReceiptsRoot() []byte {
	return e.p.ReceiptsRoot
}

// LogsBloom --
func (e executionPayloadHeader) LogsBloom() []byte {
	return e.p.LogsBloom
}

// PrevRandao --
func (e executionPayloadHeader) PrevRandao() []byte {
	return e.p.PrevRandao
}

// BlockNumber --
func (e executionPayloadHeader) BlockNumber() uint64 {
	return e.p.BlockNumber
}

// GasLimit --
func (e executionPayloadHeader) GasLimit() uint64 {
	return e.p.GasLimit
}

// GasUsed --
func (e executionPayloadHeader) GasUsed() uint64 {
	return e.p.GasUsed
}

// Timestamp --
func (e executionPayloadHeader) Timestamp() uint64 {
	return e.p.Timestamp
}

// ExtraData --
func (e executionPayloadHeader) ExtraData() []byte {
	return e.p.ExtraData
}

// BaseFeePerGas --
func (e executionPayloadHeader) BaseFeePerGas() []byte {
	return e.p.BaseFeePerGas
}

// BlockHash --
func (e executionPayloadHeader) BlockHash() []byte {
	return e.p.BlockHash
}

// Transactions --
func (executionPayloadHeader) Transactions() ([][]byte, error) {
	return nil, ErrUnsupportedGetter
}

// TransactionsRoot --
func (e executionPayloadHeader) TransactionsRoot() ([]byte, error) {
	return e.p.TransactionsRoot, nil
}

// Withdrawals --
func (e executionPayloadHeader) Withdrawals() ([]*enginev1.Withdrawal, error) {
	return nil, ErrUnsupportedGetter
}

// WithdrawalsRoot --
func (e executionPayloadHeader) WithdrawalsRoot() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

// ExcessiveDataGas --
func (e executionPayloadHeader) ExcessiveDataGas() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

// PbV2 --
func (executionPayloadHeader) PbCapella() (*enginev1.ExecutionPayloadCapella, error) {
	return nil, ErrUnsupportedGetter
}

// PbBellatrix --
func (executionPayloadHeader) PbBellatrix() (*enginev1.ExecutionPayload, error) {
	return nil, ErrUnsupportedGetter
}

// PayloadToHeader converts `payload` into execution payload header format.
func PayloadToHeader(payload interfaces.ExecutionData) (*enginev1.ExecutionPayloadHeader, error) {
	txs, err := payload.Transactions()
	if err != nil {
		return nil, err
	}
	txRoot, err := ssz.TransactionsRoot(txs)
	if err != nil {
		return nil, err
	}
	return &enginev1.ExecutionPayloadHeader{
		ParentHash:       bytesutil.SafeCopyBytes(payload.ParentHash()),
		FeeRecipient:     bytesutil.SafeCopyBytes(payload.FeeRecipient()),
		StateRoot:        bytesutil.SafeCopyBytes(payload.StateRoot()),
		ReceiptsRoot:     bytesutil.SafeCopyBytes(payload.ReceiptsRoot()),
		LogsBloom:        bytesutil.SafeCopyBytes(payload.LogsBloom()),
		PrevRandao:       bytesutil.SafeCopyBytes(payload.PrevRandao()),
		BlockNumber:      payload.BlockNumber(),
		GasLimit:         payload.GasLimit(),
		GasUsed:          payload.GasUsed(),
		Timestamp:        payload.Timestamp(),
		ExtraData:        bytesutil.SafeCopyBytes(payload.ExtraData()),
		BaseFeePerGas:    bytesutil.SafeCopyBytes(payload.BaseFeePerGas()),
		BlockHash:        bytesutil.SafeCopyBytes(payload.BlockHash()),
		TransactionsRoot: txRoot[:],
	}, nil
}

// executionPayloadCapella is a convenience wrapper around a beacon block body's execution payload data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayloadCapella struct {
	p *enginev1.ExecutionPayloadCapella
}

// WrappedExecutionPayloadCapella is a constructor which wraps a protobuf execution payload into an interface.
func WrappedExecutionPayloadCapella(p *enginev1.ExecutionPayloadCapella) (interfaces.ExecutionData, error) {
	w := executionPayloadCapella{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// IsNil checks if the underlying data is nil.
func (e executionPayloadCapella) IsNil() bool {
	return e.p == nil
}

// MarshalSSZ --
func (e executionPayloadCapella) MarshalSSZ() ([]byte, error) {
	return e.p.MarshalSSZ()
}

// MarshalSSZTo --
func (e executionPayloadCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	return e.p.MarshalSSZTo(dst)
}

// SizeSSZ --
func (e executionPayloadCapella) SizeSSZ() int {
	return e.p.SizeSSZ()
}

// UnmarshalSSZ --
func (e executionPayloadCapella) UnmarshalSSZ(buf []byte) error {
	return e.p.UnmarshalSSZ(buf)
}

// HashTreeRoot --
func (e executionPayloadCapella) HashTreeRoot() ([32]byte, error) {
	return e.p.HashTreeRoot()
}

// HashTreeRootWith --
func (e executionPayloadCapella) HashTreeRootWith(hh *fastssz.Hasher) error {
	return e.p.HashTreeRootWith(hh)
}

// Proto --
func (e executionPayloadCapella) Proto() proto.Message {
	return e.p
}

// ParentHash --
func (e executionPayloadCapella) ParentHash() []byte {
	return e.p.ParentHash
}

// FeeRecipient --
func (e executionPayloadCapella) FeeRecipient() []byte {
	return e.p.FeeRecipient
}

// StateRoot --
func (e executionPayloadCapella) StateRoot() []byte {
	return e.p.StateRoot
}

// ReceiptsRoot --
func (e executionPayloadCapella) ReceiptsRoot() []byte {
	return e.p.ReceiptsRoot
}

// LogsBloom --
func (e executionPayloadCapella) LogsBloom() []byte {
	return e.p.LogsBloom
}

// PrevRandao --
func (e executionPayloadCapella) PrevRandao() []byte {
	return e.p.PrevRandao
}

// BlockNumber --
func (e executionPayloadCapella) BlockNumber() uint64 {
	return e.p.BlockNumber
}

// GasLimit --
func (e executionPayloadCapella) GasLimit() uint64 {
	return e.p.GasLimit
}

// GasUsed --
func (e executionPayloadCapella) GasUsed() uint64 {
	return e.p.GasUsed
}

// Timestamp --
func (e executionPayloadCapella) Timestamp() uint64 {
	return e.p.Timestamp
}

// ExtraData --
func (e executionPayloadCapella) ExtraData() []byte {
	return e.p.ExtraData
}

// BaseFeePerGas --
func (e executionPayloadCapella) BaseFeePerGas() []byte {
	return e.p.BaseFeePerGas
}

// BlockHash --
func (e executionPayloadCapella) BlockHash() []byte {
	return e.p.BlockHash
}

// Transactions --
func (e executionPayloadCapella) Transactions() ([][]byte, error) {
	return e.p.Transactions, nil
}

// TransactionsRoot --
func (e executionPayloadCapella) TransactionsRoot() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

// Withdrawals --
func (e executionPayloadCapella) Withdrawals() ([]*enginev1.Withdrawal, error) {
	return e.p.Withdrawals, nil
}

// WithdrawalsRoot --
func (e executionPayloadCapella) WithdrawalsRoot() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

func (e executionPayloadCapella) ExcessiveDataGas() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

// PbV2 --
func (e executionPayloadCapella) PbCapella() (*enginev1.ExecutionPayloadCapella, error) {
	return e.p, nil
}

// PbBellatrix --
func (executionPayloadCapella) PbBellatrix() (*enginev1.ExecutionPayload, error) {
	return nil, ErrUnsupportedGetter
}

// executionPayloadHeaderCapella is a convenience wrapper around a blinded beacon block body's execution header data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayloadHeaderCapella struct {
	p *enginev1.ExecutionPayloadHeaderCapella
}

// WrappedExecutionPayloadHeaderCapella is a constructor which wraps a protobuf execution header into an interface.
func WrappedExecutionPayloadHeaderCapella(p *enginev1.ExecutionPayloadHeaderCapella) (interfaces.ExecutionData, error) {
	w := executionPayloadHeaderCapella{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// IsNil checks if the underlying data is nil.
func (e executionPayloadHeaderCapella) IsNil() bool {
	return e.p == nil
}

// MarshalSSZ --
func (e executionPayloadHeaderCapella) MarshalSSZ() ([]byte, error) {
	return e.p.MarshalSSZ()
}

// MarshalSSZTo --
func (e executionPayloadHeaderCapella) MarshalSSZTo(dst []byte) ([]byte, error) {
	return e.p.MarshalSSZTo(dst)
}

// SizeSSZ --
func (e executionPayloadHeaderCapella) SizeSSZ() int {
	return e.p.SizeSSZ()
}

// UnmarshalSSZ --
func (e executionPayloadHeaderCapella) UnmarshalSSZ(buf []byte) error {
	return e.p.UnmarshalSSZ(buf)
}

// HashTreeRoot --
func (e executionPayloadHeaderCapella) HashTreeRoot() ([32]byte, error) {
	return e.p.HashTreeRoot()
}

// HashTreeRootWith --
func (e executionPayloadHeaderCapella) HashTreeRootWith(hh *fastssz.Hasher) error {
	return e.p.HashTreeRootWith(hh)
}

// Proto --
func (e executionPayloadHeaderCapella) Proto() proto.Message {
	return e.p
}

// ParentHash --
func (e executionPayloadHeaderCapella) ParentHash() []byte {
	return e.p.ParentHash
}

// FeeRecipient --
func (e executionPayloadHeaderCapella) FeeRecipient() []byte {
	return e.p.FeeRecipient
}

// StateRoot --
func (e executionPayloadHeaderCapella) StateRoot() []byte {
	return e.p.StateRoot
}

// ReceiptsRoot --
func (e executionPayloadHeaderCapella) ReceiptsRoot() []byte {
	return e.p.ReceiptsRoot
}

// LogsBloom --
func (e executionPayloadHeaderCapella) LogsBloom() []byte {
	return e.p.LogsBloom
}

// PrevRandao --
func (e executionPayloadHeaderCapella) PrevRandao() []byte {
	return e.p.PrevRandao
}

// BlockNumber --
func (e executionPayloadHeaderCapella) BlockNumber() uint64 {
	return e.p.BlockNumber
}

// GasLimit --
func (e executionPayloadHeaderCapella) GasLimit() uint64 {
	return e.p.GasLimit
}

// GasUsed --
func (e executionPayloadHeaderCapella) GasUsed() uint64 {
	return e.p.GasUsed
}

// Timestamp --
func (e executionPayloadHeaderCapella) Timestamp() uint64 {
	return e.p.Timestamp
}

// ExtraData --
func (e executionPayloadHeaderCapella) ExtraData() []byte {
	return e.p.ExtraData
}

// BaseFeePerGas --
func (e executionPayloadHeaderCapella) BaseFeePerGas() []byte {
	return e.p.BaseFeePerGas
}

// BlockHash --
func (e executionPayloadHeaderCapella) BlockHash() []byte {
	return e.p.BlockHash
}

// Transactions --
func (executionPayloadHeaderCapella) Transactions() ([][]byte, error) {
	return nil, ErrUnsupportedGetter
}

// TransactionsRoot --
func (e executionPayloadHeaderCapella) TransactionsRoot() ([]byte, error) {
	return e.p.TransactionsRoot, nil
}

// Withdrawals --
func (e executionPayloadHeaderCapella) Withdrawals() ([]*enginev1.Withdrawal, error) {
	return nil, ErrUnsupportedGetter
}

// WithdrawalsRoot --
func (e executionPayloadHeaderCapella) WithdrawalsRoot() ([]byte, error) {
	return e.p.WithdrawalsRoot, nil
}

func (e executionPayloadHeaderCapella) ExcessiveDataGas() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

// PbV2 --
func (executionPayloadHeaderCapella) PbCapella() (*enginev1.ExecutionPayloadCapella, error) {
	return nil, ErrUnsupportedGetter
}

// PbBellatrix --
func (executionPayloadHeaderCapella) PbBellatrix() (*enginev1.ExecutionPayload, error) {
	return nil, ErrUnsupportedGetter
}

// PayloadToHeaderCapella converts `payload` into execution payload header format.
func PayloadToHeaderCapella(payload interfaces.ExecutionData) (*enginev1.ExecutionPayloadHeaderCapella, error) {
	txs, err := payload.Transactions()
	if err != nil {
		return nil, err
	}
	txRoot, err := ssz.TransactionsRoot(txs)
	if err != nil {
		return nil, err
	}
	withdrawals, err := payload.Withdrawals()
	if err != nil {
		return nil, err
	}
	withdrawalsRoot, err := ssz.WithdrawalSliceRoot(hash.CustomSHA256Hasher(), withdrawals, fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return nil, err
	}

	return &enginev1.ExecutionPayloadHeaderCapella{
		ParentHash:       bytesutil.SafeCopyBytes(payload.ParentHash()),
		FeeRecipient:     bytesutil.SafeCopyBytes(payload.FeeRecipient()),
		StateRoot:        bytesutil.SafeCopyBytes(payload.StateRoot()),
		ReceiptsRoot:     bytesutil.SafeCopyBytes(payload.ReceiptsRoot()),
		LogsBloom:        bytesutil.SafeCopyBytes(payload.LogsBloom()),
		PrevRandao:       bytesutil.SafeCopyBytes(payload.PrevRandao()),
		BlockNumber:      payload.BlockNumber(),
		GasLimit:         payload.GasLimit(),
		GasUsed:          payload.GasUsed(),
		Timestamp:        payload.Timestamp(),
		ExtraData:        bytesutil.SafeCopyBytes(payload.ExtraData()),
		BaseFeePerGas:    bytesutil.SafeCopyBytes(payload.BaseFeePerGas()),
		BlockHash:        bytesutil.SafeCopyBytes(payload.BlockHash()),
		TransactionsRoot: txRoot[:],
		WithdrawalsRoot:  withdrawalsRoot[:],
	}, nil
}

// PayloadToHeaderEIP4844 converts `payload` into execution payload header format.
func PayloadToHeaderEIP4844(payload interfaces.ExecutionData) (*enginev1.ExecutionPayloadHeader4844, error) {
	txs, err := payload.Transactions()
	if err != nil {
		return nil, err
	}
	txRoot, err := ssz.TransactionsRoot(txs)
	if err != nil {
		return nil, err
	}
	withdrawals, err := payload.Withdrawals()
	if err != nil {
		return nil, err
	}
	withdrawalsRoot, err := ssz.WithdrawalSliceRoot(hash.CustomSHA256Hasher(), withdrawals, fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return nil, err
	}
	excessDataGas, err := payload.ExcessiveDataGas()
	if err != nil {
		return nil, err
	}

	return &enginev1.ExecutionPayloadHeader4844{
		ParentHash:       bytesutil.SafeCopyBytes(payload.ParentHash()),
		FeeRecipient:     bytesutil.SafeCopyBytes(payload.FeeRecipient()),
		StateRoot:        bytesutil.SafeCopyBytes(payload.StateRoot()),
		ReceiptsRoot:     bytesutil.SafeCopyBytes(payload.ReceiptsRoot()),
		LogsBloom:        bytesutil.SafeCopyBytes(payload.LogsBloom()),
		PrevRandao:       bytesutil.SafeCopyBytes(payload.PrevRandao()),
		BlockNumber:      payload.BlockNumber(),
		GasLimit:         payload.GasLimit(),
		GasUsed:          payload.GasUsed(),
		Timestamp:        payload.Timestamp(),
		ExtraData:        bytesutil.SafeCopyBytes(payload.ExtraData()),
		BaseFeePerGas:    bytesutil.SafeCopyBytes(payload.BaseFeePerGas()),
		BlockHash:        bytesutil.SafeCopyBytes(payload.BlockHash()),
		ExcessDataGas:    bytesutil.SafeCopyBytes(excessDataGas),
		TransactionsRoot: txRoot[:],
		WithdrawalsRoot:  withdrawalsRoot[:],
	}, nil
}

// IsEmptyExecutionData checks if an execution data is empty underneath. If a single field has
// a non-zero value, this function will return false.
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
	case errors.Is(err, ErrUnsupportedGetter):
	case err != nil:
		return false, err
	default:
		if len(txs) != 0 {
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

// executionPayloadHeaderEIP4844 is a convenience wrapper around a blinded beacon block body's execution header data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayloadHeaderEIP4844 struct {
	p *enginev1.ExecutionPayloadHeader4844
}

// WrappedExecutionPayloadHeaderEIP4844 is a constructor which wraps a protobuf execution header into an interface.
func WrappedExecutionPayloadHeaderEIP4844(p *enginev1.ExecutionPayloadHeader4844) (interfaces.ExecutionData, error) {
	w := executionPayloadHeaderEIP4844{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// IsNil checks if the underlying data is nil.
func (e executionPayloadHeaderEIP4844) IsNil() bool {
	return e.p == nil
}

// MarshalSSZ --
func (e executionPayloadHeaderEIP4844) MarshalSSZ() ([]byte, error) {
	return e.p.MarshalSSZ()
}

// MarshalSSZTo --
func (e executionPayloadHeaderEIP4844) MarshalSSZTo(dst []byte) ([]byte, error) {
	return e.p.MarshalSSZTo(dst)
}

// SizeSSZ --
func (e executionPayloadHeaderEIP4844) SizeSSZ() int {
	return e.p.SizeSSZ()
}

// UnmarshalSSZ --
func (e executionPayloadHeaderEIP4844) UnmarshalSSZ(buf []byte) error {
	return e.p.UnmarshalSSZ(buf)
}

// HashTreeRoot --
func (e executionPayloadHeaderEIP4844) HashTreeRoot() ([32]byte, error) {
	return e.p.HashTreeRoot()
}

// HashTreeRootWith --
func (e executionPayloadHeaderEIP4844) HashTreeRootWith(hh *fastssz.Hasher) error {
	return e.p.HashTreeRootWith(hh)
}

// Proto --
func (e executionPayloadHeaderEIP4844) Proto() proto.Message {
	return e.p
}

// ParentHash --
func (e executionPayloadHeaderEIP4844) ParentHash() []byte {
	return e.p.ParentHash
}

// FeeRecipient --
func (e executionPayloadHeaderEIP4844) FeeRecipient() []byte {
	return e.p.FeeRecipient
}

// StateRoot --
func (e executionPayloadHeaderEIP4844) StateRoot() []byte {
	return e.p.StateRoot
}

// ReceiptsRoot --
func (e executionPayloadHeaderEIP4844) ReceiptsRoot() []byte {
	return e.p.ReceiptsRoot
}

// LogsBloom --
func (e executionPayloadHeaderEIP4844) LogsBloom() []byte {
	return e.p.LogsBloom
}

// PrevRandao --
func (e executionPayloadHeaderEIP4844) PrevRandao() []byte {
	return e.p.PrevRandao
}

// BlockNumber --
func (e executionPayloadHeaderEIP4844) BlockNumber() uint64 {
	return e.p.BlockNumber
}

// GasLimit --
func (e executionPayloadHeaderEIP4844) GasLimit() uint64 {
	return e.p.GasLimit
}

// GasUsed --
func (e executionPayloadHeaderEIP4844) GasUsed() uint64 {
	return e.p.GasUsed
}

// Timestamp --
func (e executionPayloadHeaderEIP4844) Timestamp() uint64 {
	return e.p.Timestamp
}

// ExtraData --
func (e executionPayloadHeaderEIP4844) ExtraData() []byte {
	return e.p.ExtraData
}

// BaseFeePerGas --
func (e executionPayloadHeaderEIP4844) BaseFeePerGas() []byte {
	return e.p.BaseFeePerGas
}

// BlockHash --
func (e executionPayloadHeaderEIP4844) BlockHash() []byte {
	return e.p.BlockHash
}

// Transactions --
func (executionPayloadHeaderEIP4844) Transactions() ([][]byte, error) {
	return nil, ErrUnsupportedGetter
}

// TransactionsRoot --
func (e executionPayloadHeaderEIP4844) TransactionsRoot() ([]byte, error) {
	return e.p.TransactionsRoot, nil
}

// Withdrawals --
func (e executionPayloadHeaderEIP4844) Withdrawals() ([]*enginev1.Withdrawal, error) {
	return nil, ErrUnsupportedGetter
}

// WitdrawalsRoot --
func (e executionPayloadHeaderEIP4844) WithdrawalsRoot() ([]byte, error) {
	return e.p.WithdrawalsRoot, nil
}

func (e executionPayloadHeaderEIP4844) ExcessiveDataGas() ([]byte, error) {
	return e.p.ExcessDataGas, nil
}

// PbBellatrix --
func (e executionPayloadHeaderEIP4844) PbBellatrix() (*enginev1.ExecutionPayload, error) {
	return nil, ErrUnsupportedGetter
}

// PbCapella --
func (e executionPayloadHeaderEIP4844) PbCapella() (*enginev1.ExecutionPayloadCapella, error) {
	return nil, ErrUnsupportedGetter
}

// executionPayloadEIP4844 is a convenience wrapper around a beacon block body's execution payload data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayloadEIP4844 struct {
	p *enginev1.ExecutionPayload4844
}

// WrappedExecutionPayloadEIP4844 is a constructor which wraps a protobuf execution payload into an interface.
func WrappedExecutionPayloadEIP4844(p *enginev1.ExecutionPayload4844) (interfaces.ExecutionData, error) {
	w := executionPayloadEIP4844{p: p}
	if w.IsNil() {
		return nil, ErrNilObjectWrapped
	}
	return w, nil
}

// IsNil checks if the underlying data is nil.
func (e executionPayloadEIP4844) IsNil() bool {
	return e.p == nil
}

// MarshalSSZ --
func (e executionPayloadEIP4844) MarshalSSZ() ([]byte, error) {
	return e.p.MarshalSSZ()
}

// MarshalSSZTo --
func (e executionPayloadEIP4844) MarshalSSZTo(dst []byte) ([]byte, error) {
	return e.p.MarshalSSZTo(dst)
}

// SizeSSZ --
func (e executionPayloadEIP4844) SizeSSZ() int {
	return e.p.SizeSSZ()
}

// UnmarshalSSZ --
func (e executionPayloadEIP4844) UnmarshalSSZ(buf []byte) error {
	return e.p.UnmarshalSSZ(buf)
}

// HashTreeRoot --
func (e executionPayloadEIP4844) HashTreeRoot() ([32]byte, error) {
	return e.p.HashTreeRoot()
}

// HashTreeRootWith --
func (e executionPayloadEIP4844) HashTreeRootWith(hh *fastssz.Hasher) error {
	return e.p.HashTreeRootWith(hh)
}

// Proto --
func (e executionPayloadEIP4844) Proto() proto.Message {
	return e.p
}

// ParentHash --
func (e executionPayloadEIP4844) ParentHash() []byte {
	return e.p.ParentHash
}

// FeeRecipient --
func (e executionPayloadEIP4844) FeeRecipient() []byte {
	return e.p.FeeRecipient
}

// StateRoot --
func (e executionPayloadEIP4844) StateRoot() []byte {
	return e.p.StateRoot
}

// ReceiptsRoot --
func (e executionPayloadEIP4844) ReceiptsRoot() []byte {
	return e.p.ReceiptsRoot
}

// LogsBloom --
func (e executionPayloadEIP4844) LogsBloom() []byte {
	return e.p.LogsBloom
}

// PrevRandao --
func (e executionPayloadEIP4844) PrevRandao() []byte {
	return e.p.PrevRandao
}

// BlockNumber --
func (e executionPayloadEIP4844) BlockNumber() uint64 {
	return e.p.BlockNumber
}

// GasLimit --
func (e executionPayloadEIP4844) GasLimit() uint64 {
	return e.p.GasLimit
}

// GasUsed --
func (e executionPayloadEIP4844) GasUsed() uint64 {
	return e.p.GasUsed
}

// Timestamp --
func (e executionPayloadEIP4844) Timestamp() uint64 {
	return e.p.Timestamp
}

// ExtraData --
func (e executionPayloadEIP4844) ExtraData() []byte {
	return e.p.ExtraData
}

// BaseFeePerGas --
func (e executionPayloadEIP4844) BaseFeePerGas() []byte {
	return e.p.BaseFeePerGas
}

// BlockHash --
func (e executionPayloadEIP4844) BlockHash() []byte {
	return e.p.BlockHash
}

// Transactions --
func (e executionPayloadEIP4844) Transactions() ([][]byte, error) {
	return e.p.Transactions, nil
}

// TransactionsRoot --
func (e executionPayloadEIP4844) TransactionsRoot() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

// Withdrawals --
func (e executionPayloadEIP4844) Withdrawals() ([]*enginev1.Withdrawal, error) {
	return e.p.Withdrawals, nil
}

// WithdrawalsRoot --
func (e executionPayloadEIP4844) WithdrawalsRoot() ([]byte, error) {
	return nil, ErrUnsupportedGetter
}

func (e executionPayloadEIP4844) ExcessiveDataGas() ([]byte, error) {
	return e.p.ExcessDataGas, nil
}

// PbBellatrix --
func (e executionPayloadEIP4844) PbBellatrix() (*enginev1.ExecutionPayload, error) {
	return nil, ErrUnsupportedGetter
}

// PbCapella --
func (e executionPayloadEIP4844) PbCapella() (*enginev1.ExecutionPayloadCapella, error) {
	return nil, ErrUnsupportedGetter
}
