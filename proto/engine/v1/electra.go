package enginev1

import (
	"github.com/pkg/errors"
)

type ExecutionPayloadElectra = ExecutionPayloadDeneb
type ExecutionPayloadHeaderElectra = ExecutionPayloadHeaderDeneb

var (
	drExample = &DepositRequest{}
	drSize    = drExample.SizeSSZ()
	wrExample = &WithdrawalRequest{}
	wrSize    = wrExample.SizeSSZ()
	crExample = &ConsolidationRequest{}
	crSize    = crExample.SizeSSZ()
)

func (eee *ExecutionEnvelopeElectra) GetDecodedExecutionRequests() (*ExecutionRequests, error) {
	requests := &ExecutionRequests{}

	if len(eee.ExecutionRequests) != 3 /* types of requests */ {
		return nil, errors.Errorf("invalid execution request size: %d", len(eee.ExecutionRequests))
	}

	// deposit requests
	drBytes := eee.ExecutionRequests[0]
	if len(drBytes)%drSize != 0 {
		return nil, errors.New("invalid ssz length for deposit requests")
	}
	numDepositRequests := len(drBytes) / drSize
	requests.Deposits = make([]*DepositRequest, numDepositRequests)
	for i := range requests.Deposits {
		requestBytes := drBytes[i*drSize : (i+1)*drSize]
		dr := &DepositRequest{}
		if err := dr.UnmarshalSSZ(requestBytes); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal deposit request")
		}
		requests.Deposits[i] = dr
	}

	// withdrawal requests
	wrBytes := eee.ExecutionRequests[1]
	if len(wrBytes)%wrSize != 0 {
		return nil, errors.New("invalid ssz length for withdrawal requests")
	}
	numWithdrawalRequests := len(wrBytes) / wrSize
	requests.Withdrawals = make([]*WithdrawalRequest, numWithdrawalRequests)
	for i := range requests.Withdrawals {
		requestBytes := wrBytes[i*wrSize : (i+1)*wrSize]
		wr := &WithdrawalRequest{}
		if err := wr.UnmarshalSSZ(requestBytes); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal withdrawal request")
		}
		requests.Withdrawals[i] = wr
	}

	// consolidation requests
	crBytes := eee.ExecutionRequests[2]
	if len(crBytes)%crSize != 0 {
		return nil, errors.New("invalid ssz length for consolidation requests")
	}
	numConsolidationRequests := len(crBytes) / crSize
	requests.Consolidations = make([]*ConsolidationRequest, numConsolidationRequests)
	for i := range requests.Consolidations {
		requestBytes := crBytes[i*wrSize : (i+1)*wrSize]
		cr := &ConsolidationRequest{}
		if err := cr.UnmarshalSSZ(requestBytes); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal consolidation request")
		}
		requests.Consolidations[i] = cr
	}

	return requests, nil
}

func EncodeExecutionRequests(requests *ExecutionRequests) ([][]byte, error) {
	if requests == nil {
		return nil, errors.New("invalid execution requests")
	}

	drBytes := make([]byte, len(requests.Deposits)*drSize)
	for i, deposit := range requests.Deposits {
		dr, err := deposit.MarshalSSZ()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal deposit request")
		}
		copy(drBytes[i:i+drSize], dr)
	}

	wrBytes := make([]byte, len(requests.Withdrawals)*wrSize)
	for i, withdrawal := range requests.Withdrawals {
		wr, err := withdrawal.MarshalSSZ()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal withdrawal request")
		}
		copy(wrBytes[i*wrSize:i+wrSize], wr)
	}

	crBytes := make([]byte, len(requests.Consolidations)*crSize)
	for i, consolidation := range requests.Consolidations {
		cr, err := consolidation.MarshalSSZ()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal consolidation request")
		}
		copy(crBytes[i*crSize:i+crSize], cr)
	}
	return [][]byte{drBytes, wrBytes, crBytes}, nil
}
