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

// ExecutionEngineCaller defines methods that wraps around execution engine API calls to enable other prysm services to interact with.
type ExecutionEngineCaller interface {
	PreparePayload(ctx context.Context, parentHash []byte, timeStamp uint64, random []byte, feeRecipient []byte) (uint64, error)
	GetPayload(ctx context.Context, payloadID uint64) (*ethpb.ExecutionPayload, error)
	// NotifyConsensusValidated is the wrapper on top of `CatalystClient` to abstract out `types.ConsensusValidatedParams`.
	NotifyConsensusValidated(ctx context.Context, blockHash []byte, valid bool) error
	NotifyForkChoiceValidated(ctx context.Context, headBlockHash []byte, finalizedBlockHash []byte) error
	// ExecutePayload is the wrapper on top of `CatalystClient` to abstract out `types.GenericStringResponse`.
	ExecutePayload(ctx context.Context, payload *ethpb.ExecutionPayload) error
}

// CatalystClient calls with the execution engine end points to enable consensus <-> execution interaction.
type CatalystClient interface {
	PreparePayload(ctx context.Context, params types.AssembleBlockParams) (*types.PayloadResponse, error)
	GetPayload(ctx context.Context, PayloadID hexutil.Uint64) (*types.ExecutableData, error)
	// ConsensusValidated notifies execution engine on the result of beacon state transition.
	ConsensusValidated(ctx context.Context, params types.ConsensusValidatedParams) error
	ForkchoiceUpdated(ctx context.Context, params types.ForkChoiceParams) error
	// ExecutePayload returns true if and only if input executable data is valid with respect to engine state.
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

// NotifyConsensusValidated notifies execution engine on the result of beacon state transition.
// Per definition, consensus engine must notify execution engine after `state_transition` function finishes.
// The value of valid parameters must be set as follows:
// -True if state_transition function call succeeds
// -False if state_transition function call fails
// Engine API definition:
// 	https://github.com/ethereum/consensus-specs/blob/dev/specs/merge/beacon-chain.md#notify_consensus_validated
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

// ExecutePayload executes execution payload by calling execution engine.
// Engine API definition:
// 	https://github.com/ethereum/execution-apis/blob/main/src/engine/interop/specification.md#engine_executepayload
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

	// The `respond` definition:
	// https://github.com/ethereum/execution-apis/blob/main/src/engine/interop/specification.md#returns-2
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
