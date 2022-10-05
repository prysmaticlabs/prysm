package blocks

import (
	"bytes"
	"errors"

	fastssz "github.com/prysmaticlabs/fastssz"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"google.golang.org/protobuf/proto"
)

// executionPayloadHeader is a convenience wrapper around a beacon block body's payload header data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayloadHeader struct {
	version          int
	parentHash       []byte
	feeRecipient     []byte
	stateRoot        []byte
	receiptsRoot     []byte
	logsBloom        []byte
	prevRandao       []byte
	blockNumber      uint64
	gasLimit         uint64
	gasUsed          uint64
	timestamp        uint64
	extraData        []byte
	baseFeePerGas    []byte
	blockHash        []byte
	transactionsRoot []byte
	excessDataGas    []byte
}

// IsNil checks if the underlying data is nil.
func (e *executionPayloadHeader) IsNil() bool {
	return e == nil
}

// MarshalSSZ --
func (e *executionPayloadHeader) MarshalSSZ() ([]byte, error) {
	pb, err := e.Proto()
	if err != nil {
		return []byte{}, err
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayloadHeader).MarshalSSZ()
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayloadHeader4844).MarshalSSZ()
	default:
		return []byte{}, errIncorrectPayloadVersion
	}
}

// MarshalSSZTo --
func (e *executionPayloadHeader) MarshalSSZTo(dst []byte) ([]byte, error) {
	pb, err := e.Proto()
	if err != nil {
		return []byte{}, err
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayloadHeader).MarshalSSZTo(dst)
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayloadHeader4844).MarshalSSZTo(dst)
	default:
		return []byte{}, errIncorrectPayloadVersion
	}
}

// SizeSSZ --
func (e *executionPayloadHeader) SizeSSZ() int {
	pb, err := e.Proto()
	if err != nil {
		panic(err)
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayloadHeader).SizeSSZ()
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayloadHeader4844).SizeSSZ()
	default:
		panic(errIncorrectPayloadVersion)
	}
}

// UnmarshalSSZ --
func (e *executionPayloadHeader) UnmarshalSSZ(buf []byte) error {
	var newPayload *executionPayloadHeader
	switch e.version {
	case version.Bellatrix:
		pb := &enginev1.ExecutionPayloadHeader{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newPayload, err = initPayloadHeaderFromProto(pb)
		if err != nil {
			return err
		}
	case version.EIP4844:
		pb := &enginev1.ExecutionPayloadHeader4844{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newPayload, err = initPayloadHeaderFromProto4844(pb)
		if err != nil {
			return err
		}
	default:
		return errIncorrectPayloadVersion
	}
	*e = *newPayload
	return nil
}

func (e *executionPayloadHeader) Version() int {
	return e.version
}

// HashTreeRoot --
func (e *executionPayloadHeader) HashTreeRoot() ([32]byte, error) {
	pb, err := e.Proto()
	if err != nil {
		return [32]byte{}, err
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayloadHeader).HashTreeRoot()
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayloadHeader4844).HashTreeRoot()
	default:
		return [32]byte{}, errIncorrectPayloadVersion
	}
}

// HashTreeRootWith --
func (e *executionPayloadHeader) HashTreeRootWith(h *fastssz.Hasher) error {
	pb, err := e.Proto()
	if err != nil {
		return err
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayloadHeader).HashTreeRootWith(h)
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayloadHeader4844).HashTreeRootWith(h)
	default:
		return errIncorrectPayloadVersion
	}
}

func (e *executionPayloadHeader) PbGenericPayload() (*enginev1.ExecutionPayload, error) {
	return nil, ErrUnsupportedGetter
}

func (e *executionPayloadHeader) PbEip4844Payload() (*enginev1.ExecutionPayload4844, error) {
	return nil, ErrUnsupportedGetter
}

func (e *executionPayloadHeader) PbGenericPayloadHeader() (*enginev1.ExecutionPayloadHeader, error) {
	if e.version != version.Bellatrix {
		return nil, errNotSupported("PbGenericPayloadHeader", e.version)
	}

	proto, err := e.Proto()
	if err != nil {
		return nil, err
	}
	return proto.(*enginev1.ExecutionPayloadHeader), nil
}

func (e *executionPayloadHeader) PbEip4844PayloadHeader() (*enginev1.ExecutionPayloadHeader4844, error) {
	if e.version != version.EIP4844 {
		return nil, errNotSupported("PbEip4844PayloadHeader", e.version)
	}

	proto, err := e.Proto()
	if err != nil {
		return nil, err
	}
	return proto.(*enginev1.ExecutionPayloadHeader4844), nil
}

// Proto --
func (e *executionPayloadHeader) Proto() (proto.Message, error) {
	switch e.version {
	case version.Bellatrix:
		return &enginev1.ExecutionPayloadHeader{
			ParentHash:       e.parentHash,
			FeeRecipient:     e.feeRecipient,
			StateRoot:        e.stateRoot,
			ReceiptsRoot:     e.receiptsRoot,
			LogsBloom:        e.logsBloom,
			PrevRandao:       e.prevRandao,
			BlockNumber:      e.blockNumber,
			GasLimit:         e.gasLimit,
			GasUsed:          e.gasUsed,
			Timestamp:        e.timestamp,
			ExtraData:        e.extraData,
			BaseFeePerGas:    e.baseFeePerGas,
			BlockHash:        e.blockHash,
			TransactionsRoot: e.transactionsRoot,
		}, nil
	case version.EIP4844:
		return &enginev1.ExecutionPayloadHeader4844{
			ParentHash:       e.parentHash,
			FeeRecipient:     e.feeRecipient,
			StateRoot:        e.stateRoot,
			ReceiptsRoot:     e.receiptsRoot,
			LogsBloom:        e.logsBloom,
			PrevRandao:       e.prevRandao,
			BlockNumber:      e.blockNumber,
			GasLimit:         e.gasLimit,
			GasUsed:          e.gasUsed,
			Timestamp:        e.timestamp,
			ExtraData:        e.extraData,
			BaseFeePerGas:    e.baseFeePerGas,
			BlockHash:        e.blockHash,
			TransactionsRoot: e.transactionsRoot,
			ExcessDataGas:    e.excessDataGas,
		}, nil
	default:
		return nil, errors.New("unsupported execution payload")
	}
}

// ParentHash --
func (e *executionPayloadHeader) ParentHash() []byte {
	return e.parentHash
}

// FeeRecipient --
func (e *executionPayloadHeader) FeeRecipient() []byte {
	return e.feeRecipient
}

// StateRoot --
func (e *executionPayloadHeader) StateRoot() []byte {
	return e.stateRoot
}

// ReceiptsRoot --
func (e *executionPayloadHeader) ReceiptsRoot() []byte {
	return e.receiptsRoot
}

// LogsBloom --
func (e *executionPayloadHeader) LogsBloom() []byte {
	return e.logsBloom
}

// PrevRandao --
func (e *executionPayloadHeader) PrevRandao() []byte {
	return e.prevRandao
}

// BlockNumber --
func (e *executionPayloadHeader) BlockNumber() uint64 {
	return e.blockNumber
}

// GasLimit --
func (e *executionPayloadHeader) GasLimit() uint64 {
	return e.gasLimit
}

// GasUsed --
func (e *executionPayloadHeader) GasUsed() uint64 {
	return e.gasUsed
}

// Timestamp --
func (e *executionPayloadHeader) Timestamp() uint64 {
	return e.timestamp
}

// ExtraData --
func (e *executionPayloadHeader) ExtraData() []byte {
	return e.extraData
}

// BaseFeePerGas --
func (e *executionPayloadHeader) BaseFeePerGas() []byte {
	return e.baseFeePerGas
}

// BlockHash --
func (e *executionPayloadHeader) BlockHash() []byte {
	return e.blockHash
}

// Transactions --
func (e *executionPayloadHeader) TransactionsRoot() []byte {
	return e.transactionsRoot
}

func (e *executionPayloadHeader) Transactions() ([][]byte, error) {
	return nil, ErrUnsupportedGetter
}

// ExcessDataGas --
func (e *executionPayloadHeader) ExcessDataGas() ([]byte, error) {
	switch e.version {
	case version.EIP4844:
		return e.excessDataGas, nil
	default:
		return nil, ErrUnsupportedGetter
	}
}

// IsEmptyExecutionData checks if an execution data is empty underneath. If a single field has
// a non-zero value, this function will return false.
func IsEmptyExecutionDataHeader(data interfaces.ExecutionDataHeader) (bool, error) {
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
	if !bytes.Equal(data.TransactionsRoot(), make([]byte, fieldparams.RootLength)) {
		return false, nil
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

	excessDataGas, err := data.ExcessDataGas()
	switch {
	case errors.Is(err, ErrUnsupportedGetter):
	case err != nil:
		return false, err
	default:
		if !bytes.Equal(excessDataGas, make([]byte, fieldparams.RootLength)) {
			return false, nil
		}
	}

	return true, nil
}
