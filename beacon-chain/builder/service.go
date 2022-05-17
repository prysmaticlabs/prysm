package builder

import (
	"context"
	"encoding/hex"
	"log"

	"github.com/prysmaticlabs/prysm/api/client/builder"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/network"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
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
}

func NewService(ctx context.Context, opts ...Option) (*Service, error) {
	s := &Service{
		cfg: &config{},
	}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err

		}
	}
	c, err := builder.NewClient(s.cfg.builderEndpoint.Url)
	if err != nil {
		return nil, err
	}

	h := "a0513a503d5bd6e89a144c3268e5b7e9da9dbf63df125a360e3950a7d0d67131"
	data, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	b, err := c.GetHeader(ctx, 1, bytesutil.ToBytes32(data), [48]byte{})
	if err != nil {
		return nil, err
	}
	log.Println(b)

	log.Fatal("End of test")

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

func (s *Service) RegisterValidator(context.Context, *ethpb.SignedValidatorRegistrationV1) error {
	panic("implement me")
}
