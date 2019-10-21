// Package operations defines the life-cycle of beacon block operations.
package operations

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	handler "github.com/prysmaticlabs/prysm/shared/messagehandler"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "operation")

// Pool defines an interface for fetching the list of attestations
// which have been observed by the beacon node but not yet included in
// a beacon block by a proposer.
type Pool interface {
	AttestationPool(ctx context.Context, requestedSlot uint64) ([]*ethpb.Attestation, error)
	AttestationPoolNoVerify(ctx context.Context) ([]*ethpb.Attestation, error)
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

type recentAttestationMultiMap struct {
	lock           sync.RWMutex
	slotRootMap    map[uint64][32]byte
	rootBitlistMap map[[32]byte]bitfield.Bitlist
}

func newRecentAttestationMultiMap() *recentAttestationMultiMap {
	return &recentAttestationMultiMap{
		slotRootMap:    make(map[uint64][32]byte),
		rootBitlistMap: make(map[[32]byte]bitfield.Bitlist),
	}
}

// Prune removes expired attestation references from the map.
func (r *recentAttestationMultiMap) Prune(currentSlot uint64) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for slot, root := range r.slotRootMap {
		// Block expiration period is slots_per_epoch, we'll keep references to attestations within
		// twice that range to act as a short circuit for incoming attestations that may have been
		// delayed in the network.
		if slot+(2*params.BeaconConfig().SlotsPerEpoch)+1 < currentSlot {
			delete(r.slotRootMap, slot)
			delete(r.rootBitlistMap, root)
		}
	}
}

func (r *recentAttestationMultiMap) Insert(slot uint64, root [32]byte, bitlist bitfield.Bitlist) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.slotRootMap[slot] = root
	if b, exists := r.rootBitlistMap[root]; exists {
		r.rootBitlistMap[root] = b.Or(bitlist)
	} else {
		r.rootBitlistMap[root] = bitlist
	}
}

func (r *recentAttestationMultiMap) Contains(root [32]byte, b bitfield.Bitlist) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	a, ok := r.rootBitlistMap[root]
	if !ok {
		return false
	}
	return a.Contains(b)
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
	attestationPool            map[[32]byte]*dbpb.AttestationContainer
	recentAttestationBitlist   *recentAttestationMultiMap
	attestationPoolLock        sync.Mutex
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
		attestationPool:            make(map[[32]byte]*dbpb.AttestationContainer),
		recentAttestationBitlist:   newRecentAttestationMultiMap(),
		attestationLockCache:       ccache.New(ccache.Configure()),
	}
}

// Start an beacon block operation pool service's main event loop.
func (s *Service) Start() {
	go s.removeOperations()
}

// Stop the beacon block operation pool service's main event loop
// and associated goroutines.
func (s *Service) Stop() error {
	defer s.cancel()
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
// the attestations are returned in target epoch ascending order and up to MaxAttestations
// capacity. The attestations returned will be verified against the head state up to requested slot.
// When fails attestation, the attestation will be removed from the pool.
func (s *Service) AttestationPool(ctx context.Context, requestedSlot uint64) ([]*ethpb.Attestation, error) {
	s.attestationPoolLock.Lock()
	defer s.attestationPoolLock.Unlock()

	ctx, span := trace.StartSpan(ctx, "operations.AttestationPool")
	defer span.End()

	atts := make([]*ethpb.Attestation, 0, len(s.attestationPool))

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

	var validAttsCount uint64
	for root, ac := range s.attestationPool {
		for _, att := range ac.ToAttestations() {
			if s.recentAttestationBitlist.Contains(root, att.AggregationBits) {
				continue
			}

			if _, err = blocks.ProcessAttestation(ctx, bState, att); err != nil {
				delete(s.attestationPool, root)
				continue
			}

			validAttsCount++
			// Stop the max attestation number per beacon block is reached.
			if validAttsCount == params.BeaconConfig().MaxAttestations {
				break
			}

			atts = append(atts, att)
		}
	}

	return atts, nil
}

// AttestationPoolNoVerify returns every attestation from the attestation pool.
func (s *Service) AttestationPoolNoVerify(ctx context.Context) ([]*ethpb.Attestation, error) {
	s.attestationPoolLock.Lock()
	defer s.attestationPoolLock.Unlock()

	atts := make([]*ethpb.Attestation, 0, len(s.attestationPool))

	for _, ac := range s.attestationPool {
		atts = append(atts, ac.ToAttestations()...)
	}

	return atts, nil
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
	s.attestationPoolLock.Lock()
	defer s.attestationPoolLock.Unlock()

	ctx, span := trace.StartSpan(ctx, "operations.HandleAttestation")
	defer span.End()

	attestation := message.(*ethpb.Attestation)
	root, err := ssz.HashTreeRoot(attestation.Data)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}

	if s.recentAttestationBitlist.Contains(root, attestation.AggregationBits) {
		log.Debug("Attestation aggregation bits already included recently")
		return nil
	}

	ac, ok := s.attestationPool[root]
	if !ok {
		s.attestationPool[root] = dbpb.NewContainerFromAttestations([]*ethpb.Attestation{attestation})
		return nil
	}

	// Container already has attestation(s) that fully contain the the aggregation bits of this new
	// attestation so there is nothing to insert or aggregate.
	if ac.Contains(attestation) {
		log.Debug("Attestation already fully contained in container")
		return nil
	}

	beforeAggregation := append(ac.ToAttestations(), attestation)

	// Filter any out attestation that is already fully included.
	for i, att := range beforeAggregation {
		if s.recentAttestationBitlist.Contains(root, att.AggregationBits) {
			beforeAggregation = append(beforeAggregation[:i], beforeAggregation[i+1:]...)
		}
	}

	aggregated, err := helpers.AggregateAttestations(beforeAggregation)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return err
	}

	s.attestationPool[root] = dbpb.NewContainerFromAttestations(aggregated)

	return nil
}

// removeOperations removes the processed operations from operation pool and DB.
func (s *Service) removeOperations() {
	incomingBlockSub := s.incomingProcessedBlockFeed.Subscribe(s.incomingProcessedBlock)
	defer incomingBlockSub.Unsubscribe()

	for {
		ctx := context.TODO()
		select {
		case err := <-incomingBlockSub.Err():
			log.WithError(err).Error("Subscription to incoming block sub failed")
			return
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return
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
	s.recentAttestationBitlist.Prune(block.Slot)

	for i, att := range block.Body.Attestations {
		root, err := ssz.HashTreeRoot(att.Data)
		if err != nil {
			return err
		}
		log.WithFields(logrus.Fields{
			"index":            i,
			"root":             fmt.Sprintf("%#x", root),
			"aggregation_bits": fmt.Sprintf("%8b", att.AggregationBits.Bytes()),
			"shard":            att.Data.Crosslink.Shard,
		}).Debug("block attestation")
	}
	return nil
}

// removeAttestationsFromPool removes a list of attestations from the DB
// after they have been included in a beacon block.
func (s *Service) removeAttestationsFromPool(ctx context.Context, attestations []*ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "operations.removeAttestationsFromPool")
	defer span.End()
	s.attestationPoolLock.Lock()
	defer s.attestationPoolLock.Unlock()

	for _, attestation := range attestations {
		root, err := ssz.HashTreeRoot(attestation.Data)
		if err != nil {
			return err
		}

		// TODO(1428): Update this to attestation.Slot.
		// References upstream issue https://github.com/ethereum/eth2.0-specs/pull/1428
		slot := helpers.StartSlot(attestation.Data.Target.Epoch)
		s.recentAttestationBitlist.Insert(slot, root, attestation.AggregationBits)

		log = log.WithField("root", fmt.Sprintf("%#x", root))

		ac, ok := s.attestationPool[root]
		if ok {
			atts := ac.ToAttestations()
			for i, att := range atts {
				if s.recentAttestationBitlist.Contains(root, att.AggregationBits) {
					log.Debug("deleting attestation")
					if i < len(atts)-1 {
						copy(atts[i:], atts[i+1:])
					}
					atts[len(atts)-1] = nil
					atts = atts[:len(atts)-1]
				}
			}

			if len(atts) == 0 {
				delete(s.attestationPool, root)
				continue
			}

			s.attestationPool[root] = dbpb.NewContainerFromAttestations(atts)
		} else {
			log.Debug("No attestations found with this root.")
		}
	}
	return nil
}
