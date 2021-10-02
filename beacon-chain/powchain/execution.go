package powchain

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ExecutionEngineCaller defines methods that call execution engine API to enable other prysm services to interact with.
type ExecutionEngineCaller interface {
	PreparePayload(parentHash []byte, timeStamp uint64, random []byte, feeRecipient []byte) (uint64, error)
	GetPayload(payloadID uint64) (*ethpb.ExecutionPayload, error)
	NotifyConsensusValidated(blockHash []byte, valid bool) error
	NotifyForkChoiceValidated(headBlockHash []byte, finalizedBlockHash []byte) error
	ExecutePayload(payload *ethpb.ExecutionPayload) error
}

// CatalystClient calls with the execution engine end points to enable consensus <-> execution interaction.
type CatalystClient interface {
	PreparePayload(params catalyst.AssembleBlockParams) (*catalyst.PayloadResponse, error)
	GetPayload(PayloadID hexutil.Uint64) (*catalyst.ExecutableData, error)
	ConsensusValidated(params catalyst.ConsensusValidatedParams) error
	ForkchoiceUpdated(params catalyst.ForkChoiceParams)
	ExecutePayload(params catalyst.ExecutableData) (catalyst.GenericStringResponse, error)
}

func (s *Service) PreparePayload(parentHash []byte, timeStamp uint64, random []byte, feeRecipient []byte) (uint64, error) {
	res, err := s.catalystClient.PreparePayload(catalyst.AssembleBlockParams{
		ParentHash:   common.BytesToHash(parentHash),
		Timestamp:    timeStamp,
		Random:       common.BytesToHash(random),
		FeeRecipient: common.BytesToAddress(feeRecipient),
	})
	if err != nil {
		return 0, err
	}
	return res.PayloadID, nil
}

func (s *Service) GetPayload(payloadID uint64) (*ethpb.ExecutionPayload, error) {
	payload, err := s.catalystClient.GetPayload(hexutil.Uint64(payloadID))
	if err != nil {
		return nil, err
	}
	baseFeePerGas := make([]byte, 32)
	binary.LittleEndian.PutUint64(baseFeePerGas, payload.BaseFeePerGas)
	return &ethpb.ExecutionPayload{
		ParentHash:    payload.ParentHash.Bytes(),
		Coinbase:      payload.Coinbase.Bytes(),
		StateRoot:     payload.StateRoot.Bytes(),
		ReceiptRoot:   payload.ReceiptRoot.Bytes(),
		LogsBloom:     payload.LogsBloom,
		Random:        payload.Random.Bytes(),
		BlockNumber:   payload.Number,
		GasLimit:      payload.GasLimit,
		GasUsed:       payload.GasUsed,
		Timestamp:     payload.Timestamp,
		ExtraData:     payload.ExtraData,
		BaseFeePerGas: baseFeePerGas,
		BlockHash:     payload.BlockHash.Bytes(),
		Transactions:  payload.Transactions,
	}, nil
}

func (s *Service) NotifyConsensusValidated(blockHash []byte, valid bool) error {
	status := "INVALID"
	if valid {
		status = "VALID"
	}
	return s.catalystClient.ConsensusValidated(catalyst.ConsensusValidatedParams{
		BlockHash: common.BytesToHash(blockHash),
		Status:    status,
	})
}

func (s *Service) NotifyForkChoiceValidated(headBlockHash []byte, finalizedBlockHash []byte) error {
	s.catalystClient.ForkchoiceUpdated(catalyst.ForkChoiceParams{
		HeadBlockHash:      common.BytesToHash(headBlockHash),
		FinalizedBlockHash: common.BytesToHash(finalizedBlockHash),
	})
	return nil
}

func (s *Service) ExecutePayload(payload *ethpb.ExecutionPayload) error {
	baseFeePerGas, _ := binary.Uvarint(payload.BaseFeePerGas)
	res, err := s.catalystClient.ExecutePayload(catalyst.ExecutableData{
		BlockHash:     common.BytesToHash(payload.BlockHash),
		ParentHash:    common.BytesToHash(payload.ParentHash),
		Coinbase:      common.BytesToAddress(payload.Coinbase),
		StateRoot:     common.BytesToHash(payload.StateRoot),
		ReceiptRoot:   common.BytesToHash(payload.ReceiptRoot),
		LogsBloom:     payload.LogsBloom,
		Random:        common.BytesToHash(payload.Random),
		Number:        payload.BlockNumber,
		GasLimit:      payload.GasLimit,
		GasUsed:       payload.GasUsed,
		Timestamp:     payload.Timestamp,
		ExtraData:     payload.ExtraData,
		BaseFeePerGas: baseFeePerGas, // TODO: this is suppose to be 256 bits: https://github.com/ethereum/execution-apis/blob/main/src/engine/interop/specification.md#executionpayload
		Transactions:  payload.Transactions,
	})
	if err != nil {
		return err
	}
	switch res {
	case catalyst.VALID:
		return nil
	case catalyst.INVALID:
		return errors.New("invalid payload")
	case catalyst.SYNCING:
		return errors.New("sync process is in progress")
	default:
		return errors.New("unknown execute payload response type")
	}
}
