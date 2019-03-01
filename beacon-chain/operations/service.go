// Package operations defines the life-cycle of beacon block operations.
package operations

import (
	"context"
	"fmt"
	"sort"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "operation")

// Service represents a service that handles the internal
// logic of beacon block operations.
type Service struct {
	ctx                        context.Context
	cancel                     context.CancelFunc
	beaconDB                   *db.BeaconDB
	incomingExitFeed           *event.Feed
	incomingValidatorExits     chan *pb.VoluntaryExit
	incomingAttFeed            *event.Feed
	incomingAtt                chan *pb.Attestation
	incomingProcessedBlockFeed *event.Feed
	incomingProcessedBlock     chan *pb.BeaconBlock
	error                      error
}

// Config options for the service.
type Config struct {
	BeaconDB        *db.BeaconDB
	ReceiveExitBuf  int
	ReceiveAttBuf   int
	ReceiveBlockBuf int
}

// NewOpsPoolService instantiates a new service instance that will
// be registered into a running beacon node.
func NewOpsPoolService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                        ctx,
		cancel:                     cancel,
		beaconDB:                   cfg.BeaconDB,
		incomingExitFeed:           new(event.Feed),
		incomingValidatorExits:     make(chan *pb.VoluntaryExit, cfg.ReceiveExitBuf),
		incomingAttFeed:            new(event.Feed),
		incomingAtt:                make(chan *pb.Attestation, cfg.ReceiveAttBuf),
		incomingProcessedBlockFeed: new(event.Feed),
		incomingProcessedBlock:     make(chan *pb.BeaconBlock, cfg.ReceiveBlockBuf),
	}
}

// Start an beacon block operation pool service's main event loop.
func (s *Service) Start() {
	log.Info("Starting service")
	go s.saveOperations()
	go s.removeOperations()
}

// Stop the beacon block operation pool service's main event loop
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
// The beacon block operation pool service will subscribe to this feed in order to relay incoming exits.
func (s *Service) IncomingExitFeed() *event.Feed {
	return s.incomingExitFeed
}

// IncomingAttFeed returns a feed that any service can send incoming p2p attestations into.
// The beacon block operation pool service will subscribe to this feed in order to relay incoming attestations.
func (s *Service) IncomingAttFeed() *event.Feed {
	return s.incomingAttFeed
}

// IncomingProcessedBlockFeed returns a feed that any service can send incoming p2p beacon blocks into.
// The beacon block operation pool service will subscribe to this feed in order to receive incoming beacon blocks.
func (s *Service) IncomingProcessedBlockFeed() *event.Feed {
	return s.incomingProcessedBlockFeed
}

// PendingAttestations returns the attestations that have not seen on the beacon chain, the attestations are
// returns in slot ascending order and up to MaxAttestations capacity. The attestations get
// deleted in DB after they have been retrieved.
func (s *Service) PendingAttestations() ([]*pb.Attestation, error) {
	var attestations []*pb.Attestation
	attestationsFromDB, err := s.beaconDB.Attestations()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve attestations from DB")
	}
	sort.Slice(attestationsFromDB, func(i, j int) bool {
		return attestationsFromDB[i].Data.Slot < attestationsFromDB[j].Data.Slot
	})
	for i := range attestationsFromDB {
		// Stop the max attestation number per beacon block is reached.
		if uint64(i) == params.BeaconConfig().MaxAttestations {
			break
		}
		attestations = append(attestations, attestationsFromDB[i])
	}
	return attestations, nil
}

// saveOperations saves the newly broadcasted beacon block operations
// that was received from sync service.
func (s *Service) saveOperations() {
	// TODO(1438): Add rest of operations (slashings, attestation, exists...etc)
	incomingSub := s.incomingExitFeed.Subscribe(s.incomingValidatorExits)
	defer incomingSub.Unsubscribe()
	incomingAttSub := s.incomingAttFeed.Subscribe(s.incomingAtt)
	defer incomingAttSub.Unsubscribe()

	for {
		select {
		case <-incomingSub.Err():
			log.Debug("Subscriber closed, exiting goroutine")
			return
		case <-s.ctx.Done():
			log.Debug("operations service context closed, exiting save goroutine")
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
			log.Infof("Exit request %#x saved in DB", hash)
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
			log.Infof("Attestation %#x saved in DB", hash)
		}
	}
}

// removeOperations removes the processed operations from operation pool and DB.
func (s *Service) removeOperations() {
	incomingBlockSub := s.incomingProcessedBlockFeed.Subscribe(s.incomingProcessedBlock)
	defer incomingBlockSub.Unsubscribe()

	for {
		select {
		case <-incomingBlockSub.Err():
			log.Debug("Subscriber closed, exiting goroutine")
			return
		case <-s.ctx.Done():
			log.Debug("operations service context closed, exiting remove goroutine")
			return
		// Listen for processed block from the block chain service.
		case block := <-s.incomingProcessedBlock:
			// Removes the pending attestations received from processed block body in DB.
			if err := s.removePendingAttestations(block.Body.Attestations); err != nil {
				log.Errorf("Could not remove processed attestations from DB: %v", err)
				return
			}
		}
	}
}

// removePendingAttestations removes a list of attestations from DB.
func (s *Service) removePendingAttestations(attestations []*pb.Attestation) error {
	for _, attestation := range attestations {
		if err := s.beaconDB.DeleteAttestation(attestation); err != nil {
			return err
		}
		h, err := hashutil.HashProto(attestation)
		if err != nil {
			return err
		}
		log.WithField("attestationRoot", fmt.Sprintf("0x%x", h)).Info("Attestation removed")
	}
	return nil
}
