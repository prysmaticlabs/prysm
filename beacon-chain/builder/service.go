package builder

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/client/builder"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/network"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// BlockBuilder defines the interface for interacting with the block builder
type BlockBuilder interface {
	SubmitBlindedBlock(ctx context.Context, block *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error)
	GetHeader(ctx context.Context, slot types.Slot, parentHash [32]byte, pubKey [48]byte) (*ethpb.SignedBuilderBid, error)
	Status(ctx context.Context) error
	RegisterValidator(ctx context.Context, reg *ethpb.SignedValidatorRegistrationV1) error
}

// config defines a config struct for dependencies into the service.
type config struct {
	builderEndpoint network.Endpoint
	beaconDB        db.HeadAccessDatabase
	headFetcher     blockchain.HeadFetcher
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

// SubmitBlindedBlock submits a blinded block to the builder relay network.
func (s *Service) SubmitBlindedBlock(ctx context.Context, b *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error) {
	ctx, span := trace.StartSpan(ctx, "builder.SubmitBlindedBlock")
	defer span.End()
	start := time.Now()
	defer func() {
		submitBlindedBlockLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	return s.c.SubmitBlindedBlock(ctx, b)
}

// GetHeader retrieves the header for a given slot and parent hash from the builder relay network.
func (s *Service) GetHeader(ctx context.Context, slot types.Slot, parentHash [32]byte, pubKey [48]byte) (*ethpb.SignedBuilderBid, error) {
	ctx, span := trace.StartSpan(ctx, "builder.GetHeader")
	defer span.End()
	start := time.Now()
	defer func() {
		getHeaderLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	return s.c.GetHeader(ctx, slot, parentHash, pubKey)
}

// Status retrieves the status of the builder relay network.
func (s *Service) Status(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "builder.Status")
	defer span.End()
	start := time.Now()
	defer func() {
		getStatusLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	return s.c.Status(ctx)
}

// RegisterValidator registers a validator with the builder relay network.
// It also saves the registration object to the DB.
func (s *Service) RegisterValidator(ctx context.Context, reg *ethpb.SignedValidatorRegistrationV1) error {
	ctx, span := trace.StartSpan(ctx, "builder.RegisterValidator")
	defer span.End()
	start := time.Now()
	defer func() {
		registerValidatorLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	idx, exists := s.cfg.headFetcher.HeadPublicKeyToValidatorIndex(bytesutil.ToBytes48(reg.Message.Pubkey))
	if !exists {
		return nil // If the pubkey is not found, it is not a validator. Do nothing.
	}
	if err := s.c.RegisterValidator(ctx, reg); err != nil {
		return errors.Wrap(err, "could not register validator")
	}

	return s.cfg.beaconDB.SaveRegistrationsByValidatorIDs(ctx, []types.ValidatorIndex{idx}, []*ethpb.ValidatorRegistrationV1{reg.Message})
}
