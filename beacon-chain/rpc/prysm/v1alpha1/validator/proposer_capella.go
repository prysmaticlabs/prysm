package validator

import (
	"context"

	"github.com/pkg/errors"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
)

// constructCapellaPayloadFromBellatrix returns a wrapped Cappella Execution payload with
// empty withdrawals, from a given Bellatrix execution payload
func (vs *Server) constructCapellaPayloadFromBellatrix(
	ctx context.Context, payload *enginev1.ExecutionPayload) (interfaces.ExecutionData, error) {

	head, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state")
	}

	withdrawals, err := head.ExpectedWithdrawals()
	if err != nil {
		return nil, errors.Wrap(err, "could not get expected withdrawals")
	}

	capellaPayload := &enginev1.ExecutionPayloadCapella{
		ParentHash:    payload.ParentHash,
		FeeRecipient:  payload.FeeRecipient,
		StateRoot:     payload.StateRoot,
		ReceiptsRoot:  payload.ReceiptsRoot,
		LogsBloom:     payload.LogsBloom,
		PrevRandao:    payload.PrevRandao,
		BlockNumber:   payload.BlockNumber,
		GasLimit:      payload.GasLimit,
		GasUsed:       payload.GasUsed,
		Timestamp:     payload.Timestamp,
		ExtraData:     payload.ExtraData,
		BaseFeePerGas: payload.BaseFeePerGas,
		Transactions:  payload.Transactions,
		Withdrawals:   withdrawals,
	}
	return consensusblocks.WrappedExecutionPayloadCapella(capellaPayload)
}
