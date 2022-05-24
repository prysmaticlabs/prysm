package builder

import (
	"context"

	"github.com/prysmaticlabs/prysm/api/client/builder"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/network"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time"
	"go.opencensus.io/trace"
)

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

type Service struct {
	cfg *config
	c   *builder.Client
	err error
}

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

func (s *Service) Start() {

}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) SubmitBlindedBlock(ctx context.Context, blk *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error) {
	ctx, span := trace.StartSpan(ctx, "builder.SubmitBlindedBlock")
	defer span.End()
	start := time.Now()
	defer func() {
		submitBlindedBlockLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	return s.c.SubmitBlindedBlock(ctx, blk)
}

func (s *Service) GetHeader(ctx context.Context, slot types.Slot, parentHash [32]byte, pubKey [48]byte) (*ethpb.SignedBuilderBid, error) {
	ctx, span := trace.StartSpan(ctx, "builder.GetHeader")
	defer span.End()
	start := time.Now()
	defer func() {
		getHeaderLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	return s.c.GetHeader(ctx, slot, parentHash, pubKey)
}

func (s *Service) Status() error {
	if s.cfg.builderEndpoint.Url == "" {
		return ErrNotRunning
	}
	return s.err
}

func (s *Service) RegisterValidator(ctx context.Context, reg *ethpb.SignedValidatorRegistrationV1) error {
	ctx, span := trace.StartSpan(ctx, "builder.RegisterValidator")
	defer span.End()
	start := time.Now()
	defer func() {
		registerValidatorLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	return s.c.RegisterValidator(ctx, reg)
}
