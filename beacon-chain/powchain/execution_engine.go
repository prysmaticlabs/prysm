package powchain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/catalyst/types"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

var errNoExecutionEngineConnection = errors.New("can't connect to execution engine")

// ExecutionEngineCaller defines methods that wraps around execution engine API calls to enable other prysm services to interact with.
type ExecutionEngineCaller interface {
	// PreparePayload is a wrapper on top of `CatalystClient` to abstract out `types.AssembleBlockParams`.
	PreparePayload(ctx context.Context, parentHash []byte, timeStamp uint64, random []byte, feeRecipient []byte) (uint64, error)
	// GetPayload is a wrapper on top of `CatalystClient`.
	GetPayload(ctx context.Context, payloadID uint64) (*ethpb.ExecutionPayload, error)
	// NotifyConsensusValidated is the wrapper on top of `CatalystClient` to abstract out `types.ConsensusValidatedParams`.
	NotifyConsensusValidated(ctx context.Context, blockHash []byte, valid bool) error
	// NotifyForkChoiceValidated is the wrapper on top of `CatalystClient` to abstract out `types.ConsensusValidatedParams`.
	NotifyForkChoiceValidated(ctx context.Context, headBlockHash []byte, finalizedBlockHash []byte) error
	// ExecutePayload is the wrapper on top of `CatalystClient` to abstract out `types.ForkChoiceParams`.
	ExecutePayload(ctx context.Context, payload *ethpb.ExecutionPayload) error
	LatestExecutionBlock() (*ExecutionBlock, error)
	ExecutionBlockByHash(blockHash common.Hash) (*ExecutionBlock, error)
}

// CatalystClient calls with the execution engine end points to enable consensus <-> execution interaction.
type CatalystClient interface {
	// PreparePayload initiates a process of building an execution payload on top of the execution chain tip.
	PreparePayload(ctx context.Context, params types.AssembleBlockParams) (*types.PayloadResponse, error)
	// GetPayload returns the most recent version of the execution payload that has been built since the corresponding
	// call to prepare_payload method.
	GetPayload(ctx context.Context, PayloadID hexutil.Uint64) (*types.ExecutableData, error)
	// ConsensusValidated signals execution engine on the result of beacon state transition.
	ConsensusValidated(ctx context.Context, params types.ConsensusValidatedParams) error
	// ForkchoiceUpdated signals execution engine on the fork choice updates.
	ForkchoiceUpdated(ctx context.Context, params types.ForkChoiceParams) error
	// ExecutePayload returns true if and only if input executable data is valid with respect to engine state.
	ExecutePayload(ctx context.Context, params types.ExecutableData) (types.GenericStringResponse, error)
}

// PreparePayload initiates a process of building an execution payload on top of the execution chain tip by parent hash.
// it returns an uint64 payload id that is used to obtain the execution payload in a subsequent `GetPayload` call.
// Engine API definition:
//  https://github.com/ethereum/execution-apis/blob/main/src/engine/interop/specification.md#engine_preparepayload
func (s *Service) PreparePayload(ctx context.Context, parentHash []byte, timeStamp uint64, random []byte, feeRecipient []byte) (uint64, error) {
	if s.catalystClient == nil {
		return 0, errNoExecutionEngineConnection
	}
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

// GetPayload returns the most recent version of the execution payload that has been built since the corresponding
// call to `PreparePayload` method. It returns the `ExecutionPayload` object.
// Engine API definition:
//  https://github.com/ethereum/execution-apis/blob/main/src/engine/interop/specification.md#engine_getpayload
func (s *Service) GetPayload(ctx context.Context, payloadID uint64) (*ethpb.ExecutionPayload, error) {
	if s.catalystClient == nil {
		return nil, errNoExecutionEngineConnection
	}
	ed, err := s.catalystClient.GetPayload(ctx, hexutil.Uint64(payloadID))
	if err != nil {
		return nil, err
	}
	return executableDataToExecutionPayload(ed), nil
}

// NotifyConsensusValidated notifies execution engine on the result of beacon state transition.
// Per definition, consensus engine must notify execution engine after `state_transition` function finishes.
// The value of valid parameters must be set as follows:
// -True if state_transition function call succeeds
// -False if state_transition function call fails
// Engine API definition:
// 	https://github.com/ethereum/consensus-specs/blob/dev/specs/merge/beacon-chain.md#notify_consensus_validated
func (s *Service) NotifyConsensusValidated(ctx context.Context, blockHash []byte, valid bool) error {
	if s.catalystClient == nil {
		return errNoExecutionEngineConnection
	}
	status := "INVALID"
	if valid {
		status = "VALID"
	}
	return s.catalystClient.ConsensusValidated(ctx, types.ConsensusValidatedParams{
		BlockHash: common.BytesToHash(blockHash),
		Status:    status,
	})
}

// NotifyForkChoiceValidated notifies execution engine on fork choice updates.
// Engine API definition:
// https://github.com/ethereum/execution-apis/blob/main/src/engine/interop/specification.md#engine_forkchoiceupdated
func (s *Service) NotifyForkChoiceValidated(ctx context.Context, headBlockHash []byte, finalizedBlockHash []byte) error {
	if s.catalystClient == nil {
		return errNoExecutionEngineConnection
	}
	return s.catalystClient.ForkchoiceUpdated(ctx, types.ForkChoiceParams{
		HeadBlockHash:      common.BytesToHash(headBlockHash),
		FinalizedBlockHash: common.BytesToHash(finalizedBlockHash),
	})
}

// ExecutePayload executes execution payload by calling execution engine.
// Engine API definition:
// 	https://github.com/ethereum/execution-apis/blob/main/src/engine/interop/specification.md#engine_executepayload
func (s *Service) ExecutePayload(ctx context.Context, payload *ethpb.ExecutionPayload) error {
	if s.catalystClient == nil {
		return errNoExecutionEngineConnection
	}
	res, err := s.catalystClient.ExecutePayload(ctx, executionPayloadToExecutableData(payload))
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

func executionPayloadToExecutableData(payload *ethpb.ExecutionPayload) types.ExecutableData {
	txs := make([][]byte, len(payload.Transactions))
	for i, t := range payload.Transactions {
		txs[i] = t.GetOpaqueTransaction()
	}
	baseFeePerGas := new(big.Int)
	// TODO_MERGE: The conversion from 32bytes to big int is broken. This assumes base fee per gas in single digit
	baseFeePerGas.SetBytes([]byte{payload.BaseFeePerGas[0]})

	return types.ExecutableData{
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
		Transactions:  txs,
	}
}

func executableDataToExecutionPayload(ed *types.ExecutableData) *ethpb.ExecutionPayload {
	txs := make([]*ethpb.Transaction, len(ed.Transactions))
	for i, t := range ed.Transactions {
		txs[i] = &ethpb.Transaction{
			TransactionOneof: &ethpb.Transaction_OpaqueTransaction{OpaqueTransaction: t},
		}
	}

	return &ethpb.ExecutionPayload{
		ParentHash:    bytesutil.PadTo(ed.ParentHash.Bytes(), 32),
		Coinbase:      bytesutil.PadTo(ed.Coinbase.Bytes(), 20),
		StateRoot:     bytesutil.PadTo(ed.StateRoot.Bytes(), 32),
		ReceiptRoot:   bytesutil.PadTo(ed.ReceiptRoot.Bytes(), 32),
		LogsBloom:     bytesutil.PadTo(ed.LogsBloom, 256),
		Random:        bytesutil.PadTo(ed.Random.Bytes(), 32),
		BlockNumber:   ed.Number,
		GasLimit:      ed.GasLimit,
		GasUsed:       ed.GasUsed,
		Timestamp:     ed.Timestamp,
		ExtraData:     ed.ExtraData,
		BaseFeePerGas: bytesutil.PadTo(ed.BaseFeePerGas.Bytes(), 32),
		BlockHash:     bytesutil.PadTo(ed.BlockHash.Bytes(), 32),
		Transactions:  txs,
	}
}
