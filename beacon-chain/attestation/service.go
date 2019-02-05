// Package attestation defines the life-cycle and status of single and aggregated attestation.
package attestation

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "attestation")

// Service represents a service that handles the internal
// logic of managing single and aggregated attestation.
type Service struct {
	ctx           context.Context
	cancel        context.CancelFunc
	beaconDB      *db.BeaconDB
	broadcastFeed *event.Feed
	broadcastChan chan *pb.Attestation
	incomingFeed  *event.Feed
	incomingChan  chan *pb.Attestation
	// AttestationPool is the mapping of individual
	// validator's public key to it's newest attestation.
	LatestAttestation map[[48]byte]*pb.Attestation
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
		ctx:               ctx,
		cancel:            cancel,
		beaconDB:          cfg.BeaconDB,
		broadcastFeed:     new(event.Feed),
		broadcastChan:     make(chan *pb.Attestation, cfg.BroadcastAttestationBuf),
		incomingFeed:      new(event.Feed),
		incomingChan:      make(chan *pb.Attestation, cfg.ReceiveAttestationBuf),
		LatestAttestation: make(map[[48]byte]*pb.Attestation),
	}
}

// Start an attestation service's main event loop.
func (a *Service) Start() {
	log.Info("Starting service")
	go a.attestationPool()
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

// attestationPool takes an newly received attestation from sync service
// and updates attestation pool.
func (a *Service) attestationPool() {
	incomingSub := a.incomingFeed.Subscribe(a.incomingChan)
	defer incomingSub.Unsubscribe()

	for {
		select {
		case <-a.ctx.Done():
			log.Debug("Attestation pool closed, exiting goroutine")
			return
		// Listen for a newly received incoming attestation from the sync service.
		case attestation := <-a.incomingChan:
			enc, err := proto.Marshal(attestation)
			if err != nil {
				log.Errorf("Could not marshal incoming attestation to bytes: %v", err)
				continue
			}
			h := hashutil.Hash(enc)

			if err := a.updateLatestAttestation(attestation); err != nil {
				log.Errorf("Could not update attestation pool: %v", err)
				continue
			}

			log.Debugf("Updated attestation pool for attestation %#x", h)
		}
	}
}

// updateLatestAttestation inputs an new attestation and checks whether
// the attesters who submitted this attestation with the higher slot number
// have been noted in the attestation pool. If not, it updates the
// attestation pool with attester's public key to attestation.
func (a *Service) updateLatestAttestation(attestation *pb.Attestation) error {
	// Potential improvement, instead of getting the state,
	// we could get a mapping of validator index to public key.
	state, err := a.beaconDB.State()
	if err != nil {
		return err
	}

	// The participation bitfield from attestation is represented in bytes,
	// here we multiply by 8 to get an accurate validator count in bits.
	bitfield := attestation.AggregationBitfield
	totalBits := len(bitfield) * 8

	// Check each bit of participation bitfield to find out which
	// attester has submitted new attestation.
	// This is has O(n) run time and could be optimized down the line.
	for i := 0; i < totalBits; i++ {
		bitSet, err := bitutil.CheckBit(bitfield, i)
		if err != nil {
			return err
		}

		if !bitSet {
			continue
		}
		// If the attestation came from this attester.
		pubkey := bytesutil.ToBytes48(state.ValidatorRegistry[i].Pubkey)
		newAttestationSlot := attestation.Data.Slot
		currentAttestationSlot := uint64(0)
		if _, exists := a.LatestAttestation[pubkey]; exists {
			currentAttestationSlot = a.LatestAttestation[pubkey].Data.Slot
		}
		// If the attestation is newer than this attester's one in pool.
		if newAttestationSlot > currentAttestationSlot {
			a.LatestAttestation[pubkey] = attestation
		}

	}
	return nil
}
