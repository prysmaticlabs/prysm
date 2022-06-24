package testing

import (
	"context"

	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// MockBuilderService to mock builder.
type MockBuilderService struct {
	HasConfigured         bool
	Payload               *v1.ExecutionPayload
	ErrSubmitBlindedBlock error
	Bid                   *ethpb.SignedBuilderBid
	ErrGetHeader          error
	ErrStatus             error
	ErrRegisterValidator  error
}

// Configured for mocking.
func (s *MockBuilderService) Configured() bool {
	return s.HasConfigured
}

// SubmitBlindedBlock for mocking.
func (s *MockBuilderService) SubmitBlindedBlock(context.Context, *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error) {
	return s.Payload, s.ErrSubmitBlindedBlock
}

// GetHeader for mocking.
func (s *MockBuilderService) GetHeader(context.Context, types.Slot, [32]byte, [48]byte) (*ethpb.SignedBuilderBid, error) {
	return s.Bid, s.ErrGetHeader
}

// Status for mocking.
func (s *MockBuilderService) Status() error {
	return s.ErrStatus
}

// RegisterValidator for mocking.
func (s *MockBuilderService) RegisterValidator(context.Context, []*ethpb.SignedValidatorRegistrationV1) error {
	return s.ErrRegisterValidator
}
