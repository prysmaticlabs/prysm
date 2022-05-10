package builder

import (
	"context"

	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type BlockBuilder interface {
	SubmitBlindedBlock(ctx context.Context, block *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error)
	GetHeader(ctx context.Context, slot types.Slot, parentHash [32]byte, pubKey [48]byte) (*ethpb.SignedBuilderBid, error)
	Status() error
	RegisterValidator(ctx context.Context, regs []*ethpb.SignedValidatorRegistrationV1) error
}

type Service struct{}

func NewService(ctx context.Context, opts ...Option) (*Service, error) {
	s := &Service{}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Service) Start() {}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) SubmitBlindedBlock(context.Context, *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error) {
	panic("implement me")
}

func (s *Service) GetHeader(context.Context, types.Slot, [32]byte, [48]byte) (*ethpb.SignedBuilderBid, error) {
	panic("implement me")
}

func (s *Service) Status() error {
	panic("implement me")
}

func (s *Service) RegisterValidator(context.Context, []*ethpb.SignedValidatorRegistrationV1) error {
	panic("implement me")
}
