package powchain

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/catalyst"
)

// ExecutionEngineRunner runs the methods that call execution engine API to enable prysm services interact with
type ExecutionEngineRunner interface {}


// ExecutionEngineCaller calls with the execution engine end points to enable consensus <-> execution interaction.
type ExecutionEngineCaller interface {
	PreparePayload(params catalyst.AssembleBlockParams) (*catalyst.PayloadResponse, error)
	GetPayload(PayloadID hexutil.Uint64) (*catalyst.ExecutableData, error)
	ConsensusValidated(params catalyst.ConsensusValidatedParams) error
	ForkchoiceUpdated(params catalyst.ForkChoiceParams)
	ExecutePayload(params catalyst.ExecutableData) (catalyst.GenericStringResponse, error)
}
