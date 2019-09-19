// Package operations defines the life-cycle of beacon block operations.
package operations

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	handler "github.com/prysmaticlabs/prysm/shared/messagehandler"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "operation")

// Pool defines an interface for fetching the list of attestations
// which have been observed by the beacon node but not yet included in
// a beacon block by a proposer.
type Pool interface {
	AttestationPool(ctx context.Context, requestedSlot uint64) ([]*ethpb.Attestation, error)
}

// Handler defines an interface for a struct equipped for receiving block operations.
type Handler interface {
	HandleAttestation(context.Context, proto.Message) error
}

// OperationFeeds inteface defines the informational feeds from the operations
// service.
type OperationFeeds interface {
	IncomingProcessedBlockFeed() *event.Feed
}

// Service represents a service that handles the internal
// logic of beacon block operations.
type Service struct {
	ctx                        context.Context
	cancel                     context.CancelFunc
	beaconDB                   db.Database
	incomingProcessedBlockFeed *event.Feed
	incomingProcessedBlock     chan *ethpb.BeaconBlock
	error                      error
	attestationLockCache       *ccache.Cache
}

// Config options for the service.
type Config struct {
	BeaconDB db.Database
}

// NewService instantiates a new operation service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                        ctx,
		cancel:                     cancel,
		beaconDB:                   cfg.BeaconDB,
		incomingProcessedBlockFeed: new(event.Feed),
		incomingProcessedBlock:     make(chan *ethpb.BeaconBlock, params.BeaconConfig().DefaultBufferSize),
		attestationLockCache:       ccache.New(ccache.Configure()),
	}
}

// Start an beacon block operation pool service's main event loop.
func (s *Service) Start() {
	log.Info("Starting service")
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

// IncomingProcessedBlockFeed returns a feed that any service can send incoming p2p beacon blocks into.
// The beacon block operation pool service will subscribe to this feed in order to receive incoming beacon blocks.
func (s *Service) IncomingProcessedBlockFeed() *event.Feed {
	return s.incomingProcessedBlockFeed
}

// retrieves a lock for the specific data root.
func (s *Service) retrieveLock(key [32]byte) *sync.Mutex {
	keyString := string(key[:])
	mutex := &sync.Mutex{}
	item := s.attestationLockCache.Get(keyString)
	if item == nil {
		s.attestationLockCache.Set(keyString, mutex, 5*time.Minute)
		return mutex
	}
	if item.Expired() {
		s.attestationLockCache.Set(keyString, mutex, 5*time.Minute)
		item.Release()
		return mutex
	}
	return item.Value().(*sync.Mutex)
}

// AttestationPool returns the attestations that have not seen on the beacon chain,
// the attestations are returned in slot ascending order and up to MaxAttestations
// capacity. The attestations get deleted in DB after they have been retrieved.
func (s *Service) AttestationPool(ctx context.Context, requestedSlot uint64) ([]*ethpb.Attestation, error) {
	var attestations []*ethpb.Attestation
	atts, err := s.beaconDB.Attestations(ctx, nil /*filter*/)
	if err != nil {
		return nil, errors.New("could not retrieve attestations from DB")
	}
	bState, err := s.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, errors.New("could not retrieve attestations from DB")
	}

	if bState.Slot < requestedSlot {
		bState, err = state.ProcessSlots(ctx, bState, requestedSlot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process slots up to %d", requestedSlot)
		}
	}

	sort.Slice(atts, func(i, j int) bool {
		return atts[i].Data.Target.Epoch < atts[j].Data.Target.Epoch
	})

	var validAttsCount uint64
	for _, att := range atts {
		if _, err = blocks.ProcessAttestation(bState, att); err != nil {
			hash, err := ssz.HashTreeRoot(att)
			if err != nil {
				return nil, err
			}
			if err := s.beaconDB.DeleteAttestation(ctx, hash); err != nil {
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

// HandleValidatorExits processes a validator exit operation.
func (s *Service) HandleValidatorExits(ctx context.Context, message proto.Message) error {
	ctx, span := trace.StartSpan(ctx, "operations.HandleValidatorExits")
	defer span.End()

	exit := message.(*ethpb.VoluntaryExit)
	hash, err := hashutil.HashProto(exit)
	if err != nil {
		return err
	}
	if err := s.beaconDB.SaveVoluntaryExit(ctx, exit); err != nil {
		return err
	}
	log.WithField("hash", fmt.Sprintf("%#x", hash)).Info("Exit request saved in DB")
	return nil
}

// HandleAttestation processes a received attestation message.
func (s *Service) HandleAttestation(ctx context.Context, message proto.Message) error {
	ctx, span := trace.StartSpan(ctx, "operations.HandleAttestation")
	defer span.End()

	attestation := message.(*ethpb.Attestation)
	root, err := ssz.HashTreeRoot(attestation.Data)
	if err != nil {
		return err
	}

	lock := s.retrieveLock(root)
	lock.Lock()
	defer lock.Unlock()

	bState, err := s.beaconDB.HeadState(ctx)
	if err != nil {
		return err
	}

	attestationSlot := attestation.Data.Target.Epoch * params.BeaconConfig().SlotsPerEpoch
	if attestationSlot > bState.Slot {
		bState, err = state.ProcessSlots(ctx, bState, attestationSlot)
		if err != nil {
			return err
		}
	}

	incomingAttBits := attestation.AggregationBits
	if s.beaconDB.HasAttestation(ctx, root) {
		dbAtt, err := s.beaconDB.Attestation(ctx, root)
		if err != nil {
			return err
		}

		if !dbAtt.AggregationBits.Contains(incomingAttBits) {
			newAggregationBits := dbAtt.AggregationBits.Or(incomingAttBits)
			incomingAttSig, err := bls.SignatureFromBytes(attestation.Signature)
			if err != nil {
				return err
			}
			dbSig, err := bls.SignatureFromBytes(dbAtt.Signature)
			if err != nil {
				return err
			}
			aggregatedSig := bls.AggregateSignatures([]*bls.Signature{dbSig, incomingAttSig})
			dbAtt.Signature = aggregatedSig.Marshal()
			dbAtt.AggregationBits = newAggregationBits
			if err := s.beaconDB.SaveAttestation(ctx, dbAtt); err != nil {
				return err
			}
		} else {
			return nil
		}
	} else {
		if err := s.beaconDB.SaveAttestation(ctx, attestation); err != nil {
			return err
		}
	}
	return nil
}

// removeOperations removes the processed operations from operation pool and DB.
func (s *Service) removeOperations() {
	incomingBlockSub := s.incomingProcessedBlockFeed.Subscribe(s.incomingProcessedBlock)
	defer incomingBlockSub.Unsubscribe()

	for {
		ctx := context.TODO()
		select {
		case <-incomingBlockSub.Err():
			log.Debug("Subscriber closed, exiting goroutine")
		case <-s.ctx.Done():
			log.Debug("operations service context closed, exiting remove goroutine")
		// Listen for processed block from the block chain service.
		case block := <-s.incomingProcessedBlock:
			handler.SafelyHandleMessage(ctx, s.handleProcessedBlock, block)
		}
	}
}

func (s *Service) handleProcessedBlock(ctx context.Context, message proto.Message) error {
	block := message.(*ethpb.BeaconBlock)
	// Removes the attestations from the pool that have been included
	// in the received block.
	if err := s.removeAttestationsFromPool(ctx, block.Body.Attestations); err != nil {
		return errors.Wrap(err, "could not remove processed attestations from DB")
	}
	return nil
}

// removeAttestationsFromPool removes a list of attestations from the DB
// after they have been included in a beacon block.
func (s *Service) removeAttestationsFromPool(ctx context.Context, attestations []*ethpb.Attestation) error {
	for _, attestation := range attestations {
		root, err := ssz.HashTreeRoot(attestation.Data)
		if err != nil {
			return err
		}

		if s.beaconDB.HasAttestation(ctx, root) {
			if err := s.beaconDB.DeleteAttestation(ctx, root); err != nil {
				return err
			}
			log.WithField("root", fmt.Sprintf("%#x", root)).Debug("Attestation removed from pool")
		}
	}
	return nil
}
