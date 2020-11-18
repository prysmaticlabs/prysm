package slashingprotection

import (
	"context"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/validator/db"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "slashing-protection")

// Protector interface defines a struct which provides methods
// for validator slashing protection.
type Protector interface {
	IsSlashableAttestation(ctx context.Context, attestation *ethpb.IndexedAttestation) bool
	CommitAttestation(ctx context.Context, attestation *ethpb.IndexedAttestation) bool
	IsSlashableBlock(ctx context.Context, blockHeader *ethpb.BeaconBlockHeader) bool
	CommitBlock(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) (bool, error)
	shared.Service
}

// Service to manage validator slashing protection. Local slashing
// protection is mandatory at runtime but remote protection is optional.
type Service struct {
	ctx             context.Context
	cancel          context.CancelFunc
	remoteProtector Protector
	validatorDB     db.Database
}

// Config for the slashing protection service.
type Config struct {
	SlasherEndpoint            string
	CertFlag                   string
	GrpcMaxCallRecvMsgSizeFlag int
	GrpcRetriesFlag            uint
	GrpcRetryDelay             time.Duration
	GrpcHeadersFlag            string
}

// NewService creates a new validator service for the service
// registry.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	srv := &Service{
		ctx:    ctx,
		cancel: cancel,
	}
	if cfg.SlasherEndpoint != "" {
		rp, err := NewRemoteProtector(ctx, cfg)
		if err != nil {
			return nil, err
		}
		srv.remoteProtector = rp
	}
	return srv, nil
}

// Start the slasher protection service.
func (s *Service) Start() {
	if s.remoteProtector != nil {
		s.remoteProtector.Start()
	}
}

// Stop --
func (s *Service) Stop() error {
	s.cancel()
	log.Info("Stopping slashing protection service")
	if s.remoteProtector != nil {
		return s.remoteProtector.Stop()
	}
	return nil
}

// Status --
func (s *Service) Status() error {
	if s.remoteProtector != nil {
		return s.remoteProtector.Status()
	}
	return nil
}
