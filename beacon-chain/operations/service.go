// Package operations defines the life-cycle of beacon block operations.
package operations

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "operation")

// Service represents a service that handles the internal
// logic of beacon block operations.
type Service struct {
	ctx                    context.Context
	cancel                 context.CancelFunc
	beaconDB               *db.BeaconDB
	incomingExitFeed       *event.Feed
	incomingValidatorExits chan *pb.Exit
	incomingAttFeed        *event.Feed
	incomingAtt            chan *pb.Attestation
	error                  error
}

// Config options for the service.
type Config struct {
	BeaconDB       *db.BeaconDB
	ReceiveExitBuf int
	ReceiveAttBuf  int
}

// NewOperationService instantiates a new service instance that will
// be registered into a running beacon node.
func NewOperationService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                    ctx,
		cancel:                 cancel,
		beaconDB:               cfg.BeaconDB,
		incomingExitFeed:       new(event.Feed),
		incomingValidatorExits: make(chan *pb.Exit, cfg.ReceiveExitBuf),
		incomingAttFeed:        new(event.Feed),
		incomingAtt:            make(chan *pb.Attestation, cfg.ReceiveAttBuf),
	}
}

// Start an beacon block operation service's main event loop.
func (s *Service) Start() {
	log.Info("Starting service")
	go s.saveOperations()
}

// Stop the beacon block operation service's main event loop
// and associated goroutines.
func (s *Service) Stop() error {
	defer s.cancel()
	log.Info("Stopping service")
	return nil
}

// Status returns the current service error if there's any.
func (s *Service) Status() error {
	if s.error != nil {
		return s.error
	}
	return nil
}

// IncomingExitFeed returns a feed that any service can send incoming p2p exits object into.
// The beacon block operation service will subscribe to this feed in order to relay incoming exits.
func (s *Service) IncomingExitFeed() *event.Feed {
	return s.incomingExitFeed
}

// IncomingAttFeed returns a feed that any service can send incoming p2p attestations into.
// The beacon block operation service will subscribe to this feed in order to relay incoming attestations.
func (s *Service) IncomingAttFeed() *event.Feed {
	return s.incomingAttFeed
}

// saveOperations saves the newly broadcasted beacon block operations
// that was received from sync service.
func (s *Service) saveOperations() {
	// TODO(1438): Add rest of operations (slashings, attestation, exists...etc)
	incomingSub := s.incomingExitFeed.Subscribe(s.incomingValidatorExits)
	defer incomingSub.Unsubscribe()

	for {
		select {
		case <-incomingSub.Err():
			log.Debug("Subscriber closed, exiting goroutine")
			return
		case <-s.ctx.Done():
			log.Debug("Beacon block operations service context closed, exiting goroutine")
			return
		// Listen for a newly received incoming exit from the sync service.
		case exit := <-s.incomingValidatorExits:
			hash, err := hashutil.HashProto(exit)
			if err != nil {
				log.Errorf("Could not hash exit req proto: %v", err)
				continue
			}
			if err := s.beaconDB.SaveExit(exit); err != nil {
				log.Errorf("Could not save exit request: %v", err)
				continue
			}
			log.Debugf("Exit request %#x saved in db", hash)
		case attestation := <-s.incomingAtt:
			hash, err := hashutil.HashProto(attestation)
			if err != nil {
				log.Errorf("Could not hash attestation proto: %v", err)
				continue
			}
			if err := s.beaconDB.SaveAttestation(attestation); err != nil {
				log.Errorf("Could not save attestation: %v", err)
				continue
			}
			log.Debugf("Attestation %#x saved in db", hash)
		}

	}
}
