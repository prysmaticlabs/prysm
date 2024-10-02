package enginev1

import (
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/config/params"
)

type ExecutionPayloadElectra = ExecutionPayloadDeneb
type ExecutionPayloadHeaderElectra = ExecutionPayloadHeaderDeneb

func (eb *ExecutionBundleElectra) GetDecodedExecutionRequests() (*ExecutionRequests, error) {
	if len(eb.ExecutionRequests) == 0 {
		return nil, errors.New("no execution requests found")
	}
	er := &ExecutionRequests{}
	if err := processRequestBytes(eb.ExecutionRequests, er); err != nil {
		return nil, err
	}

	if err := verifyExecutionRequests(er); err != nil {
		return nil, err
	}
	return er, nil
}

func verifyExecutionRequests(er *ExecutionRequests) error {
	if uint64(len(er.Deposits)) > params.BeaconConfig().MaxDepositRequestsPerPayload {
		return fmt.Errorf("too many deposits requested: received %d, expected %d", len(er.Deposits), params.BeaconConfig().MaxDepositRequestsPerPayload)
	}
	if uint64(len(er.Withdrawals)) > params.BeaconConfig().MaxWithdrawalsPerPayload {
		return fmt.Errorf("too many withdrawals requested: received %d, expected %d", len(er.Withdrawals), params.BeaconConfig().MaxWithdrawalRequestsPerPayload)
	}
	if uint64(len(er.Consolidations)) > params.BeaconConfig().MaxConsolidationsRequestsPerPayload {
		return fmt.Errorf("too many consolidations requested: received %d, expected %d", len(er.Consolidations), params.BeaconConfig().MaxConsolidationsRequestsPerPayload)
	}
	return nil
}

func processRequestBytes(
	requestBytes []byte,
	requests *ExecutionRequests,
) error {
	if len(requestBytes) == 0 {
		return nil // No more requests to process
	}

	requestType := requestBytes[0]
	remainingBytes := requestBytes[1:]

	switch requestType {
	case 0:
		dr := &DepositRequest{}
		if len(remainingBytes) < dr.SizeSSZ() {
			return fmt.Errorf("invalid deposit request size: returned %d, expected %d", len(remainingBytes), dr.SizeSSZ())
		}
		drBytes := remainingBytes[:dr.SizeSSZ()]
		remainingBytes = remainingBytes[dr.SizeSSZ():]
		if err := dr.UnmarshalSSZ(drBytes); err != nil {
			return errors.Wrap(err, "failed to unmarshal deposit request")
		}
		requests.Deposits = append(requests.Deposits, dr)
	case 1:
		wr := &WithdrawalRequest{}
		if len(remainingBytes) < wr.SizeSSZ() {
			return fmt.Errorf("invalid withdrawal request size: returned %d, expected %d", len(remainingBytes), wr.SizeSSZ())
		}
		wrBytes := remainingBytes[:wr.SizeSSZ()]
		remainingBytes = remainingBytes[wr.SizeSSZ():]
		if err := wr.UnmarshalSSZ(wrBytes); err != nil {
			return errors.Wrap(err, "failed to unmarshal withdrawal request")
		}
		requests.Withdrawals = append(requests.Withdrawals, wr)
	case 2:
		cr := &ConsolidationRequest{}
		if len(remainingBytes) < cr.SizeSSZ() {
			return fmt.Errorf("invalid consolidation request size: returned %d, expected %d", len(remainingBytes), cr.SizeSSZ())
		}
		crBytes := remainingBytes[:cr.SizeSSZ()]
		remainingBytes = remainingBytes[cr.SizeSSZ():]
		if err := cr.UnmarshalSSZ(crBytes); err != nil {
			return errors.Wrap(err, "failed to unmarshal consolidation request")
		}
		requests.Consolidations = append(requests.Consolidations, cr)
	default:
		return errors.New("invalid execution request type")
	}

	// Recursive call with the remaining bytes
	return processRequestBytes(remainingBytes, requests)
}

func EncodeExecutionRequests(eb *ExecutionRequests) (common.Hash, error) {
	var executionRequestBytes []byte
	depositBytes, err := encodeExecutionRequestsByType(0, eb.Deposits)
	if err != nil {
		return common.Hash{}, errors.Wrap(err, "failed to encode deposit requests")
	}
	withdrawalBytes, err := encodeExecutionRequestsByType(1, eb.Withdrawals)
	if err != nil {
		return common.Hash{}, errors.Wrap(err, "failed to encode withdrawal requests")
	}
	consolidationBytes, err := encodeExecutionRequestsByType(2, eb.Consolidations)
	if err != nil {
		return common.Hash{}, errors.Wrap(err, "failed to encode consolidation requests")
	}
	executionRequestBytes = append(executionRequestBytes, depositBytes...)
	executionRequestBytes = append(executionRequestBytes, withdrawalBytes...)
	executionRequestBytes = append(executionRequestBytes, consolidationBytes...)
	return sha256.Sum256(executionRequestBytes), nil
}

type marshalSSZable interface {
	MarshalSSZ() ([]byte, error)
}

func encodeExecutionRequestsByType[T marshalSSZable](executionRequestType int, requests []T) ([]byte, error) {
	var executionRequestBytes []byte
	for _, er := range requests {
		ssz, err := er.MarshalSSZ()
		if err != nil {
			return nil, err
		}
		executionRequestBytes = append(executionRequestBytes, byte(executionRequestType))
		executionRequestBytes = append(executionRequestBytes, ssz...)
	}
	return executionRequestBytes, nil
}
