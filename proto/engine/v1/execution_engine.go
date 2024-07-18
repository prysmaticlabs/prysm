package enginev1

import "github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"

type Cloneable[T any] interface {
	Copy() T
}

// CopySlice copies the contents of a slice of pointers to a new slice.
func CopySlice[T any, C Cloneable[T]](original []C) []T {
	// Create a new slice with the same length as the original
	newSlice := make([]T, len(original))
	for i := 0; i < len(newSlice); i++ {
		newSlice[i] = original[i].Copy()
	}
	return newSlice
}

func (w *Withdrawal) Copy() *Withdrawal {
	if w == nil {
		return nil
	}

	return &Withdrawal{
		Index:          w.Index,
		ValidatorIndex: w.ValidatorIndex,
		Address:        bytesutil.SafeCopyBytes(w.Address),
		Amount:         w.Amount,
	}
}

func (d *DepositRequest) Copy() *DepositRequest {
	if d == nil {
		return nil
	}
	return &DepositRequest{
		Pubkey:                bytesutil.SafeCopyBytes(d.Pubkey),
		WithdrawalCredentials: bytesutil.SafeCopyBytes(d.WithdrawalCredentials),
		Amount:                d.Amount,
		Signature:             bytesutil.SafeCopyBytes(d.Signature),
		Index:                 d.Index,
	}
}

func (wr *WithdrawalRequest) Copy() *WithdrawalRequest {
	if wr == nil {
		return nil
	}
	return &WithdrawalRequest{
		SourceAddress:   bytesutil.SafeCopyBytes(wr.SourceAddress),
		ValidatorPubkey: bytesutil.SafeCopyBytes(wr.ValidatorPubkey),
		Amount:          wr.Amount,
	}
}

func (cr *ConsolidationRequest) Copy() *ConsolidationRequest {
	if cr == nil {
		return nil
	}
	return &ConsolidationRequest{
		SourceAddress: bytesutil.SafeCopyBytes(cr.SourceAddress),
		SourcePubkey:  bytesutil.SafeCopyBytes(cr.SourcePubkey),
		TargetPubkey:  bytesutil.SafeCopyBytes(cr.TargetPubkey),
	}
}

func (payload *ExecutionPayloadElectra) Copy() *ExecutionPayloadElectra {
	if payload == nil {
		return nil
	}
	return &ExecutionPayloadElectra{
		ParentHash:            bytesutil.SafeCopyBytes(payload.ParentHash),
		FeeRecipient:          bytesutil.SafeCopyBytes(payload.FeeRecipient),
		StateRoot:             bytesutil.SafeCopyBytes(payload.StateRoot),
		ReceiptsRoot:          bytesutil.SafeCopyBytes(payload.ReceiptsRoot),
		LogsBloom:             bytesutil.SafeCopyBytes(payload.LogsBloom),
		PrevRandao:            bytesutil.SafeCopyBytes(payload.PrevRandao),
		BlockNumber:           payload.BlockNumber,
		GasLimit:              payload.GasLimit,
		GasUsed:               payload.GasUsed,
		Timestamp:             payload.Timestamp,
		ExtraData:             bytesutil.SafeCopyBytes(payload.ExtraData),
		BaseFeePerGas:         bytesutil.SafeCopyBytes(payload.BaseFeePerGas),
		BlockHash:             bytesutil.SafeCopyBytes(payload.BlockHash),
		Transactions:          bytesutil.SafeCopy2dBytes(payload.Transactions),
		Withdrawals:           CopySlice(payload.Withdrawals),
		BlobGasUsed:           payload.BlobGasUsed,
		ExcessBlobGas:         payload.ExcessBlobGas,
		DepositRequests:       CopySlice(payload.DepositRequests),
		WithdrawalRequests:    CopySlice(payload.WithdrawalRequests),
		ConsolidationRequests: CopySlice(payload.ConsolidationRequests),
	}
}
