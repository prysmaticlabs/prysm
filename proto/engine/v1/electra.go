package enginev1

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
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

const (
	depositRequestType       = 0
	withdrawalRequestType    = 1
	consolidationRequestType = 2
)

func (ebe *ExecutionBundleElectra) GetDecodedExecutionRequests() (*ExecutionRequests, error) {
	requests := &ExecutionRequests{}

	for i := range ebe.ExecutionRequests {
		requestType := ebe.ExecutionRequests[i][0]
		requestListInSSZBytes := ebe.ExecutionRequests[i][1:]
		switch requestType {
		case depositRequestType:
			drs, err := unmarshalItems(requestListInSSZBytes, drSize, func() *DepositRequest { return &DepositRequest{} })
			if err != nil {
				return nil, err
			}
			requests.Deposits = drs
		case withdrawalRequestType:
			wrs, err := unmarshalItems(requestListInSSZBytes, wrSize, func() *WithdrawalRequest { return &WithdrawalRequest{} })
			if err != nil {
				return nil, err
			}
			requests.Withdrawals = wrs
		case consolidationRequestType:
			crs, err := unmarshalItems(requestListInSSZBytes, crSize, func() *ConsolidationRequest { return &ConsolidationRequest{} })
			if err != nil {
				return nil, err
			}
			requests.Consolidations = crs
		}
	}

	return requests, nil
}

func EncodeExecutionRequests(requests *ExecutionRequests) ([]hexutil.Bytes, error) {
	if requests == nil {
		return nil, errors.New("invalid execution requests")
	}

	var requestsData []hexutil.Bytes

	if len(requests.Deposits) > 0 {
		drBytes, err := marshalItems(requests.Deposits)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal deposit requests")
		}
		requestData := []byte{0}
		requestData = append(requestData, drBytes...)
		requestsData = append(requestsData, requestData)
	}
	if len(requests.Withdrawals) > 0 {
		wrBytes, err := marshalItems(requests.Withdrawals)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal withdrawal requests")
		}
		requestData := []byte{1}
		requestData = append(requestData, wrBytes...)
		requestsData = append(requestsData, requestData)
	}
	if len(requests.Consolidations) > 0 {
		crBytes, err := marshalItems(requests.Consolidations)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal consolidation requests")
		}
		requestData := []byte{2}
		requestData = append(requestData, crBytes...)
		requestsData = append(requestsData, requestData)
	}

	return requestsData, nil
}

type sszUnmarshaler interface {
	UnmarshalSSZ([]byte) error
}

type sszMarshaler interface {
	MarshalSSZTo(buf []byte) ([]byte, error)
	SizeSSZ() int
}

func marshalItems[T sszMarshaler](items []T) ([]byte, error) {
	if len(items) == 0 {
		return []byte{}, nil
	}
	size := items[0].SizeSSZ()
	buf := make([]byte, 0, size*len(items))
	var err error
	for i, item := range items {
		buf, err = item.MarshalSSZTo(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal item at index %d: %w", i, err)
		}
	}
	return buf, nil
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
