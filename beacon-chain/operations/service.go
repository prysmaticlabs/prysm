// Package operations defines the life-cycle of beacon block operations.
package operations

import (
	"context"
	"fmt"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	handler "github.com/prysmaticlabs/prysm/shared/messagehandler"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "operation")

// OperationFeeds inteface defines the informational feeds from the operations
// service.
type OperationFeeds interface {
	IncomingAttFeed() *event.Feed
	IncomingExitFeed() *event.Feed
	IncomingProcessedBlockFeed() *event.Feed
}

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
	p2p                        p2p.Broadcaster
	error                      error
}

// Config options for the service.
type Config struct {
	BeaconDB *db.BeaconDB
	P2P      p2p.Broadcaster
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
		incomingValidatorExits:     make(chan *pb.VoluntaryExit, params.BeaconConfig().DefaultBufferSize),
		incomingAttFeed:            new(event.Feed),
		incomingAtt:                make(chan *pb.Attestation, params.BeaconConfig().DefaultBufferSize),
		incomingProcessedBlockFeed: new(event.Feed),
		incomingProcessedBlock:     make(chan *pb.BeaconBlock, params.BeaconConfig().DefaultBufferSize),
		p2p:                        cfg.P2P,
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
func (s *Service) PendingAttestations(ctx context.Context) ([]*pb.Attestation, error) {
	var attestations []*pb.Attestation
	attestationsFromDB, err := s.beaconDB.Attestations()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve attestations from DB")
	}
	state, err := s.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve attestations from DB")
	}
	sort.Slice(attestationsFromDB, func(i, j int) bool {
		return attestationsFromDB[i].Data.Slot < attestationsFromDB[j].Data.Slot
	})
	var validAttsCount uint64
	for _, att := range attestationsFromDB {
		// Delete the attestation if the attestation is one epoch older than head state,
		// we don't want to pass these attestations to RPC for proposer to include.
		if att.Data.Slot+params.BeaconConfig().SlotsPerEpoch <= state.Slot {
			if err := s.beaconDB.DeleteAttestation(att); err != nil {
				return nil, err
			}
			continue
		}

		validAttsCount++
		// Stop the max attestation number per beacon block is reached.
		if validAttsCount == params.BeaconConfig().MaxAttestations {
			break
		}
		attestations = append(attestations, att)
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
			handler.SafelyHandleMessage(s.ctx, s.HandleValidatorExits, exit)
		case attestation := <-s.incomingAtt:
			handler.SafelyHandleMessage(s.ctx, s.HandleAttestations, attestation)
		}
	}
}

// HandleValidatorExits processes a validator exit operation.
func (s *Service) HandleValidatorExits(ctx context.Context, message proto.Message) error {
	ctx, span := trace.StartSpan(ctx, "operations.HandleValidatorExits")
	defer span.End()

	exit := message.(*pb.VoluntaryExit)
	hash, err := hashutil.HashProto(exit)
	if err != nil {
		return err
	}
	if err := s.beaconDB.SaveExit(ctx, exit); err != nil {
		return err
	}
	log.WithField("hash", fmt.Sprintf("%#x", hash)).Info("Exit request saved in DB")
	return nil
}

// HandleAttestations processes a received attestation message.
func (s *Service) HandleAttestations(ctx context.Context, message proto.Message) error {
	ctx, span := trace.StartSpan(ctx, "operations.HandleAttestations")
	defer span.End()

	attestation := message.(*pb.Attestation)
	hash, err := hashutil.HashProto(attestation)
	if err != nil {
		return err
	}
	if s.beaconDB.HasAttestation(hash) {
		return nil
	}
	if err := s.beaconDB.SaveAttestation(ctx, attestation); err != nil {
		return err
	}
	return nil
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
			handler.SafelyHandleMessage(s.ctx, s.handleProcessedBlock, block)
			// Removes the pending attestations received from processed block body in DB.
			if err := s.removePendingAttestations(block.Body.Attestations); err != nil {
				log.Errorf("Could not remove processed attestations from DB: %v", err)
				return
			}
			if err := s.removeEpochOldAttestations(block.Slot); err != nil {
				log.Errorf("Could not remove old attestations from DB at slot %d: %v", block.Slot, err)
				return
			}
		}
	}
}

func (s *Service) handleProcessedBlock(_ context.Context, message proto.Message) error {
	block := message.(*pb.BeaconBlock)
	// Removes the pending attestations received from processed block body in DB.
	if err := s.removePendingAttestations(block.Body.Attestations); err != nil {
		return fmt.Errorf("could not remove processed attestations from DB: %v", err)
	}
	return nil
}

// removePendingAttestations removes a list of attestations from DB.
func (s *Service) removePendingAttestations(attestations []*pb.Attestation) error {
	for _, attestation := range attestations {
		hash, err := hashutil.HashProto(attestation)
		if err != nil {
			return err
		}
		if s.beaconDB.HasAttestation(hash) {
			if err := s.beaconDB.DeleteAttestation(attestation); err != nil {
				return err
			}
			log.WithField("slot", attestation.Data.Slot-params.BeaconConfig().GenesisSlot).Debug("Attestation removed")
		}
	}
	return nil
}

// removeEpochOldAttestations removes attestations that's older than one epoch length from current slot.
func (s *Service) removeEpochOldAttestations(slot uint64) error {
	attestations, err := s.beaconDB.Attestations()
	if err != nil {
		return err
	}
	for _, a := range attestations {
		// Remove attestation from DB if it's one epoch older than slot.
		if slot-params.BeaconConfig().SlotsPerEpoch >= a.Data.Slot {
			if err := s.beaconDB.DeleteAttestation(a); err != nil {
				return err
			}
		}
	}
	return nil
}
