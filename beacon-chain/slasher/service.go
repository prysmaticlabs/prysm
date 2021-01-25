// Package slasher --
package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/event"
)

type ServiceConfig struct {
	*Config
	IndexedAttsFeed *event.Feed
	Database        db.Database
}

// Service --
type Service struct {
	cfg             *Config
	serviceCfg      *ServiceConfig
	indexedAttsChan chan *ethpb.IndexedAttestation
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewService --
func NewService(ctx context.Context, srvCfg *ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		cfg:             srvCfg.Config,
		serviceCfg:      srvCfg,
		indexedAttsChan: make(chan *ethpb.IndexedAttestation, 1),
		ctx:             ctx,
		cancel:          cancel,
	}, nil
}

// Start --
func (s *Service) Start() {
	s.receiveAttestations(s.ctx)
}

// Stop --
func (s *Service) Stop() error {
	s.cancel()
	return nil
}

// Status --
func (s *Service) Status() error {
	return nil
}
