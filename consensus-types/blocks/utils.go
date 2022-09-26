package blocks

import (
	"github.com/pkg/errors"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
)

func ExtractExecutionDataFromHeader(executionPayloadHeader interfaces.ExecutionPayloadHeader) (interfaces.ExecutionData, error) {
	var wrappedHeader interfaces.ExecutionData
	var err error

	switch concreteHeader := executionPayloadHeader.(type) {
	case *enginev1.ExecutionPayloadHeader:
		wrappedHeader, err = WrappedExecutionPayloadHeader(concreteHeader)
	case *enginev1.ExecutionPayloadHeader4844:
		wrappedHeader, err = WrappedExecutionPayloadHeader4844(concreteHeader)
	default:
		return nil, errors.New("Unknown execution payload header type")
	}

	return wrappedHeader, err
}
