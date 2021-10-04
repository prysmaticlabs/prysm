package powchain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/catalyst/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ExecutionEngineCaller defines methods that call execution engine API to enable other prysm services to interact with.
type ExecutionEngineCaller interface {
	PreparePayload(ctx context.Context, parentHash []byte, timeStamp uint64, random []byte, feeRecipient []byte) (uint64, error)
	GetPayload(ctx context.Context, payloadID uint64) (*ethpb.ExecutionPayload, error)
	NotifyConsensusValidated(ctx context.Context, blockHash []byte, valid bool) error
	NotifyForkChoiceValidated(ctx context.Context, headBlockHash []byte, finalizedBlockHash []byte) error
	ExecutePayload(ctx context.Context, payload *ethpb.ExecutionPayload) error
}

// CatalystClient calls with the execution engine end points to enable consensus <-> execution interaction.
type CatalystClient interface {
	PreparePayload(ctx context.Context, params types.AssembleBlockParams) (*types.PayloadResponse, error)
	GetPayload(ctx context.Context, PayloadID hexutil.Uint64) (*types.ExecutableData, error)
	ConsensusValidated(ctx context.Context, params types.ConsensusValidatedParams) error
	ForkchoiceUpdated(ctx context.Context, params types.ForkChoiceParams) error
	ExecutePayload(ctx context.Context, params types.ExecutableData) (types.GenericStringResponse, error)
}

func (s *Service) PreparePayload(ctx context.Context, parentHash []byte, timeStamp uint64, random []byte, feeRecipient []byte) (uint64, error) {
	res, err := s.catalystClient.PreparePayload(ctx, types.AssembleBlockParams{
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

func (s *Service) GetPayload(ctx context.Context, payloadID uint64) (*ethpb.ExecutionPayload, error) {
	payload, err := s.catalystClient.GetPayload(ctx, hexutil.Uint64(payloadID))
	if err != nil {
		return nil, err
	}
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
		BaseFeePerGas: payload.BaseFeePerGas.Bytes(),
		BlockHash:     payload.BlockHash.Bytes(),
		Transactions:  payload.Transactions,
	}, nil
}

func (s *Service) NotifyConsensusValidated(ctx context.Context, blockHash []byte, valid bool) error {
	status := "INVALID"
	if valid {
		status = "VALID"
	}
	return s.catalystClient.ConsensusValidated(ctx, types.ConsensusValidatedParams{
		BlockHash: common.BytesToHash(blockHash),
		Status:    status,
	})
}

func (s *Service) NotifyForkChoiceValidated(ctx context.Context, headBlockHash []byte, finalizedBlockHash []byte) error {
	return s.catalystClient.ForkchoiceUpdated(ctx, types.ForkChoiceParams{
		HeadBlockHash:      common.BytesToHash(headBlockHash),
		FinalizedBlockHash: common.BytesToHash(finalizedBlockHash),
	})
}

func (s *Service) ExecutePayload(ctx context.Context, payload *ethpb.ExecutionPayload) error {
	baseFeePerGas := new(big.Int)
	baseFeePerGas.SetBytes(payload.BaseFeePerGas)
	res, err := s.catalystClient.ExecutePayload(ctx, types.ExecutableData{
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
		BaseFeePerGas: baseFeePerGas,
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
