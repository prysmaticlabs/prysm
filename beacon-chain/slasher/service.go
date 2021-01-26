// Package slasher --
package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/event"
)

// ServiceConfig for the slasher service in the beacon node.
// This struct allows us to specify required dependencies and
// parameters for slasher to function as needed.
type ServiceConfig struct {
	*Parameters
	IndexedAttsFeed *event.Feed
	Database        db.Database
}

// Service defining a slasher implementation as part of
// the beacon node, able to detect eth2 slashable offenses.
type Service struct {
	params          *Parameters
	serviceCfg      *ServiceConfig
	indexedAttsChan chan *ethpb.IndexedAttestation
	ctx             context.Context
	cancel          context.CancelFunc
}

// New instantiates a new slasher from configuration values.
func New(ctx context.Context, srvCfg *ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		params:          srvCfg.Parameters,
		serviceCfg:      srvCfg,
		indexedAttsChan: make(chan *ethpb.IndexedAttestation, 1),
		ctx:             ctx,
		cancel:          cancel,
	}, nil
}

// Start listening for received indexed attestations and blocks
// and perform slashing detection on them.
func (s *Service) Start() {
	s.receiveAttestations(s.ctx)
}

// Stop the slasher service.
func (s *Service) Stop() error {
	s.cancel()
	return nil
}

// Status of the slasher service.
func (s *Service) Status() error {
	return nil
}
