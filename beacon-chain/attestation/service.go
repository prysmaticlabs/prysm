// Package attestation defines the life-cycle and status of single and aggregated attestation.
package attestation

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "attestation")

// Service represents a service that handles the internal
// logic of managing aggregated attestation.
type Service struct {
	ctx           context.Context
	cancel        context.CancelFunc
	beaconDB      *db.BeaconDB
	broadcastFeed *event.Feed
	broadcastChan chan *pb.Attestation
	incomingFeed  *event.Feed
	incomingChan  chan *pb.Attestation
}

// Config options for the service.
type Config struct {
	BeaconDB                *db.BeaconDB
	ReceiveAttestationBuf   int
	BroadcastAttestationBuf int
}

// NewAttestationService instantiates a new service instance that will
// be registered into a running beacon node.
func NewAttestationService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:           ctx,
		cancel:        cancel,
		beaconDB:      cfg.BeaconDB,
		broadcastFeed: new(event.Feed),
		broadcastChan: make(chan *pb.Attestation, cfg.BroadcastAttestationBuf),
		incomingFeed:  new(event.Feed),
		incomingChan:  make(chan *pb.Attestation, cfg.ReceiveAttestationBuf),
	}
}

// Start an attestation service's main event loop.
func (a *Service) Start() {
	log.Info("Starting service")
	go a.aggregateAttestations()
}

// Stop the Attestation service's main event loop and associated goroutines.
func (a *Service) Stop() error {
	defer a.cancel()
	log.Info("Stopping service")
	return nil
}

// Status always returns nil.
// TODO(1201): Add service health checks.
func (a *Service) Status() error {
	return nil
}

// IncomingAttestationFeed returns a feed that any service can send incoming p2p attestations into.
// The attestation service will subscribe to this feed in order to relay incoming attestations.
func (a *Service) IncomingAttestationFeed() *event.Feed {
	return a.incomingFeed
}

// aggregateAttestations aggregates the newly broadcasted attestation that was
// received from sync service.
func (a *Service) aggregateAttestations() {
	incomingSub := a.incomingFeed.Subscribe(a.incomingChan)
	defer incomingSub.Unsubscribe()

	for {
		select {
		case <-a.ctx.Done():
			log.Debug("Attestation service context closed, exiting goroutine")
			return
		// Listen for a newly received incoming attestation from the sync service.
		case attestation := <-a.incomingChan:
			enc, err := proto.Marshal(attestation)
			if err != nil {
				log.Errorf("Could not marshal incoming attestation to bytes: %v", err)
				continue
			}
			h := hashutil.Hash(enc)
			if err := a.beaconDB.SaveAttestation(attestation); err != nil {
				log.Errorf("Could not save attestation: %v", err)
				continue
			}

			log.Debugf("Forwarding aggregated attestation %#x to proposers through RPC", h)
		}
	}
}
