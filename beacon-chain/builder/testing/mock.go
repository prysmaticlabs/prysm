package testing

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/client/builder"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// MockBuilderService to mock builder.
type MockBuilderService struct {
	HasConfigured         bool
	Payload               *v1.ExecutionPayload
	PayloadCapella        *v1.ExecutionPayloadCapella
	ErrSubmitBlindedBlock error
	Bid                   *ethpb.SignedBuilderBid
	ErrGetHeader          error
	ErrRegisterValidator  error
}

// Configured for mocking.
func (s *MockBuilderService) Configured() bool {
	return s.HasConfigured
}

// SubmitBlindedBlock for mocking.
func (s *MockBuilderService) SubmitBlindedBlock(_ context.Context, _ interfaces.SignedBeaconBlock) (interfaces.ExecutionData, error) {
	if s.Payload != nil {
		w, err := blocks.WrappedExecutionPayload(s.Payload)
		if err != nil {
			return nil, errors.Wrap(err, "could not wrap payload")
		}
		return w, s.ErrSubmitBlindedBlock
	}
	w, err := blocks.WrappedExecutionPayloadCapella(s.PayloadCapella)
	if err != nil {
		return nil, errors.Wrap(err, "could not wrap capella payload")
	}
	return w, s.ErrSubmitBlindedBlock
}

// GetHeader for mocking.
func (s *MockBuilderService) GetHeader(context.Context, primitives.Slot, [32]byte, [48]byte) (builder.SignedBid, error) {
	w, err := builder.WrappedSignedBuilderBid(s.Bid)
	if err != nil {
		return nil, errors.Wrap(err, "could not wrap bid")
	}
	return w, s.ErrGetHeader
}

// RegisterValidator for mocking.
func (s *MockBuilderService) RegisterValidator(context.Context, []*ethpb.SignedValidatorRegistrationV1) error {
	return s.ErrRegisterValidator
}
