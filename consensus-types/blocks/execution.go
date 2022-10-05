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
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"google.golang.org/protobuf/proto"
)

// executionPayload is a convenience wrapper around a beacon block body's execution payload data structure
// This wrapper allows us to conform to a common interface so that beacon
// blocks for future forks can also be applied across Prysm without issues.
type executionPayload struct {
	version       int
	parentHash    []byte
	feeRecipient  []byte
	stateRoot     []byte
	receiptsRoot  []byte
	logsBloom     []byte
	prevRandao    []byte
	blockNumber   uint64
	gasLimit      uint64
	gasUsed       uint64
	timestamp     uint64
	extraData     []byte
	baseFeePerGas []byte
	blockHash     []byte
	transactions  [][]byte
	excessDataGas []byte
}

// IsNil checks if the underlying data is nil.
func (e *executionPayload) IsNil() bool {
	return e == nil
}

// MarshalSSZ --
func (e *executionPayload) MarshalSSZ() ([]byte, error) {
	pb, err := e.Proto()
	if err != nil {
		return []byte{}, err
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayload).MarshalSSZ()
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayload4844).MarshalSSZ()
	default:
		return []byte{}, errIncorrectPayloadVersion
	}
}

// MarshalSSZTo --
func (e *executionPayload) MarshalSSZTo(dst []byte) ([]byte, error) {
	pb, err := e.Proto()
	if err != nil {
		return []byte{}, err
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayload).MarshalSSZTo(dst)
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayload4844).MarshalSSZTo(dst)
	default:
		return []byte{}, errIncorrectPayloadVersion
	}
}

// SizeSSZ --
func (e *executionPayload) SizeSSZ() int {
	pb, err := e.Proto()
	if err != nil {
		panic(err)
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayload).SizeSSZ()
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayload4844).SizeSSZ()
	default:
		panic(errIncorrectPayloadVersion)
	}
}

// UnmarshalSSZ --
func (e *executionPayload) UnmarshalSSZ(buf []byte) error {
	var newPayload *executionPayload
	switch e.version {
	case version.Bellatrix:
		pb := &enginev1.ExecutionPayload{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newPayload, err = initPayloadFromProto(pb)
		if err != nil {
			return err
		}
	case version.EIP4844:
		pb := &enginev1.ExecutionPayload4844{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newPayload, err = initPayloadFromProto4844(pb)
		if err != nil {
			return err
		}
	default:
		return errIncorrectPayloadVersion
	}
	*e = *newPayload
	return nil
}

func (e *executionPayload) Version() int {
	return e.version
}

// HashTreeRoot --
func (e *executionPayload) HashTreeRoot() ([32]byte, error) {
	pb, err := e.Proto()
	if err != nil {
		return [32]byte{}, err
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayload).HashTreeRoot()
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayload4844).HashTreeRoot()
	default:
		return [32]byte{}, errIncorrectPayloadVersion
	}
}

// HashTreeRootWith --
func (e *executionPayload) HashTreeRootWith(h *fastssz.Hasher) error {
	pb, err := e.Proto()
	if err != nil {
		return err
	}
	switch e.version {
	case version.Bellatrix:
		return pb.(*enginev1.ExecutionPayload).HashTreeRootWith(h)
	case version.EIP4844:
		return pb.(*enginev1.ExecutionPayload4844).HashTreeRootWith(h)
	default:
		return errIncorrectPayloadVersion
	}
}

func (e *executionPayload) PbGenericPayload() (*enginev1.ExecutionPayload, error) {
	if e.version != version.Bellatrix {
		return nil, errNotSupported("PbGenericPayload", e.version)
	}
	proto, err := e.Proto()
	if err != nil {
		return nil, err
	}
	return proto.(*enginev1.ExecutionPayload), nil
}

func (e *executionPayload) PbEip4844Payload() (*enginev1.ExecutionPayload4844, error) {
	if e.version != version.EIP4844 {
		return nil, errNotSupported("PbEip4844Payload", e.version)
	}
	proto, err := e.Proto()
	if err != nil {
		return nil, err
	}
	return proto.(*enginev1.ExecutionPayload4844), nil
}

// Proto --
func (e *executionPayload) Proto() (proto.Message, error) {
	switch e.version {
	case version.Bellatrix:
		return &enginev1.ExecutionPayload{
			ParentHash:    e.parentHash,
			FeeRecipient:  e.feeRecipient,
			StateRoot:     e.stateRoot,
			ReceiptsRoot:  e.receiptsRoot,
			LogsBloom:     e.logsBloom,
			PrevRandao:    e.prevRandao,
			BlockNumber:   e.blockNumber,
			GasLimit:      e.gasLimit,
			GasUsed:       e.gasUsed,
			Timestamp:     e.timestamp,
			ExtraData:     e.extraData,
			BaseFeePerGas: e.baseFeePerGas,
			BlockHash:     e.blockHash,
			Transactions:  e.transactions,
		}, nil
	case version.EIP4844:
		return &enginev1.ExecutionPayload4844{
			ParentHash:    e.parentHash,
			FeeRecipient:  e.feeRecipient,
			StateRoot:     e.stateRoot,
			ReceiptsRoot:  e.receiptsRoot,
			LogsBloom:     e.logsBloom,
			PrevRandao:    e.prevRandao,
			BlockNumber:   e.blockNumber,
			GasLimit:      e.gasLimit,
			GasUsed:       e.gasUsed,
			Timestamp:     e.timestamp,
			ExtraData:     e.extraData,
			BaseFeePerGas: e.baseFeePerGas,
			BlockHash:     e.blockHash,
			Transactions:  e.transactions,
			ExcessDataGas: e.excessDataGas,
		}, nil
	default:
		return nil, errors.New("unsupported execution payload")
	}
}

// ParentHash --
func (e *executionPayload) ParentHash() []byte {
	return e.parentHash
}

// FeeRecipient --
func (e *executionPayload) FeeRecipient() []byte {
	return e.feeRecipient
}

// StateRoot --
func (e *executionPayload) StateRoot() []byte {
	return e.stateRoot
}

// ReceiptsRoot --
func (e *executionPayload) ReceiptsRoot() []byte {
	return e.receiptsRoot
}

// LogsBloom --
func (e *executionPayload) LogsBloom() []byte {
	return e.logsBloom
}

// PrevRandao --
func (e *executionPayload) PrevRandao() []byte {
	return e.prevRandao
}

// BlockNumber --
func (e *executionPayload) BlockNumber() uint64 {
	return e.blockNumber
}

// GasLimit --
func (e *executionPayload) GasLimit() uint64 {
	return e.gasLimit
}

// GasUsed --
func (e *executionPayload) GasUsed() uint64 {
	return e.gasUsed
}

// Timestamp --
func (e *executionPayload) Timestamp() uint64 {
	return e.timestamp
}

// ExtraData --
func (e *executionPayload) ExtraData() []byte {
	return e.extraData
}

// BaseFeePerGas --
func (e *executionPayload) BaseFeePerGas() []byte {
	return e.baseFeePerGas
}

// BlockHash --
func (e *executionPayload) BlockHash() []byte {
	return e.blockHash
}

// Transactions --
func (e executionPayload) Transactions() ([][]byte, error) {
	return e.transactions, nil
}

// ExcessDataGas --
func (e *executionPayload) ExcessDataGas() ([]byte, error) {
	switch e.version {
	case version.EIP4844:
		return e.excessDataGas, nil
	default:
		return nil, ErrUnsupportedGetter
	}
}

// PayloadToHeader converts `payload` into execution payload header format.
func PayloadToHeader(payload interfaces.ExecutionData) (interfaces.ExecutionDataHeader, error) {
	var txRoot [32]byte
	// HACK: We can sidestep an invalid getters call for Transactions() if we know we're dealing with an actual payload header
	if h, ok := payload.(*executionPayloadHeader); ok {
		txRoot = bytesutil.ToBytes32(h.transactionsRoot)
	} else {
		txs, err := payload.Transactions()
		if err != nil {
			return nil, err
		}
		txRoot, err = ssz.TransactionsRoot(txs)
		if err != nil {
			return nil, err
		}
	}
	var i interface{}
	switch payload.Version() {
	case version.Bellatrix:
		i = &enginev1.ExecutionPayloadHeader{
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
		}
	case version.EIP4844:
		excessDataGas, err := payload.ExcessDataGas()
		if err != nil {
			return nil, err
		}
		i = &enginev1.ExecutionPayloadHeader4844{
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
			ExcessDataGas:    bytesutil.SafeCopyBytes(excessDataGas),
		}
	default:
		return nil, errors.New("unsupported execution payload")
	}
	return NewExecutionDataHeader(i)
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
