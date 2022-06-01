package builder

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/client/builder"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/network"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// BlockBuilder defines the interface for interacting with the block builder
type BlockBuilder interface {
	SubmitBlindedBlock(ctx context.Context, block *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error)
	GetHeader(ctx context.Context, slot types.Slot, parentHash [32]byte, pubKey [48]byte) (*ethpb.SignedBuilderBid, error)
	Status() error
	RegisterValidator(ctx context.Context, reg *ethpb.SignedValidatorRegistrationV1) error
}

// config defines a config struct for dependencies into the service.
type config struct {
	builderEndpoint network.Endpoint
}

// Service defines a service that provides a client for interacting with the beacon chain and MEV relay network.
type Service struct {
	cfg *config
	c   *builder.Client
}

// NewService instantiates a new service.
func NewService(ctx context.Context, opts ...Option) (*Service, error) {
	s := &Service{}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	if s.cfg.builderEndpoint.Url != "" {
		c, err := builder.NewClient(s.cfg.builderEndpoint.Url)
		if err != nil {
			return nil, err
		}
		s.c = c
	}
	return s, nil
}

// Start initializes the service.
func (*Service) Start() {}

// Stop halts the service.
func (*Service) Stop() error {
	return nil
}

// SubmitBlindedBlock is currently a stub.
func (*Service) SubmitBlindedBlock(context.Context, *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error) {
	return nil, errors.New("not implemented")
}

// GetHeader is currently a stub.
func (*Service) GetHeader(context.Context, types.Slot, [32]byte, [48]byte) (*ethpb.SignedBuilderBid, error) {
	return nil, errors.New("not implemented")
}

// Status is currently a stub.
func (*Service) Status() error {
	return errors.New("not implemented")
}

// RegisterValidator is currently a stub.
func (*Service) RegisterValidator(context.Context, *ethpb.SignedValidatorRegistrationV1) error {
	return errors.New("not implemented")
}
