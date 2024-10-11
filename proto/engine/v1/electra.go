package enginev1

import (
	"fmt"

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

func (eee *ExecutionBundleElectra) GetDecodedExecutionRequests() (*ExecutionRequests, error) {
	requests := &ExecutionRequests{}

	if len(eee.ExecutionRequests) != 3 /* types of requests */ {
		return nil, errors.Errorf("invalid execution request size: %d", len(eee.ExecutionRequests))
	}

	// deposit requests
	drs, err := unmarshalItems(eee.ExecutionRequests[0], drSize, func() *DepositRequest { return &DepositRequest{} })
	if err != nil {
		return nil, err
	}
	requests.Deposits = drs

	// withdrawal requests
	wrs, err := unmarshalItems(eee.ExecutionRequests[1], wrSize, func() *WithdrawalRequest { return &WithdrawalRequest{} })
	if err != nil {
		return nil, err
	}
	requests.Withdrawals = wrs

	// consolidation requests
	crs, err := unmarshalItems(eee.ExecutionRequests[2], crSize, func() *ConsolidationRequest { return &ConsolidationRequest{} })
	if err != nil {
		return nil, err
	}
	requests.Consolidations = crs

	return requests, nil
}

func EncodeExecutionRequests(requests *ExecutionRequests) ([][]byte, error) {
	if requests == nil {
		return nil, errors.New("invalid execution requests")
	}

	drBytes, err := marshalItems(requests.Deposits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal deposit requests")
	}

	wrBytes, err := marshalItems(requests.Withdrawals)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal withdrawal requests")
	}

	crBytes, err := marshalItems(requests.Consolidations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal consolidation requests")
	}
	return [][]byte{drBytes, wrBytes, crBytes}, nil
}

type sszUnmarshaler interface {
	UnmarshalSSZ([]byte) error
}

type sszMarshaler interface {
	MarshalSSZ() ([]byte, error)
}

func marshalItems[T sszMarshaler](items []T) ([]byte, error) {
	var totalSize int
	marshaledItems := make([][]byte, len(items))
	for i, item := range items {
		bytes, err := item.MarshalSSZ()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal item at index %d: %w", i, err)
		}
		marshaledItems[i] = bytes
		totalSize += len(bytes)
	}

	result := make([]byte, totalSize)

	for i, bytes := range marshaledItems {
		copy(result[i*len(bytes):(i+1)*len(bytes)], bytes)
	}
	return result, nil
}

// Generic function to unmarshal items
func unmarshalItems[T sszUnmarshaler](data []byte, itemSize int, newItem func() T) ([]T, error) {
	if len(data)%itemSize != 0 {
		return nil, fmt.Errorf("invalid data length: data size (%d) is not a multiple of item size (%d)", len(data), itemSize)
	}
	numItems := len(data) / itemSize
	items := make([]T, numItems)
	for i := range items {
		itemBytes := data[i*itemSize : (i+1)*itemSize]
		item := newItem()
		if err := item.UnmarshalSSZ(itemBytes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal item at index %d: %w", i, err)
		}
		items[i] = item
	}
	return items, nil
}
