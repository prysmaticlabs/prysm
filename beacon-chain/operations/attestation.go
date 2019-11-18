package operations

import (
	"context"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Pool defines an interface for fetching the list of attestations
// which have been observed by the beacon node but not yet included in
// a beacon block by a proposer.
type Pool interface {
	AttestationPool(ctx context.Context, requestedSlot uint64) ([]*ethpb.Attestation, error)
	AttestationPoolNoVerify(ctx context.Context) ([]*ethpb.Attestation, error)
	AttestationPoolForForkchoice(ctx context.Context) ([]*ethpb.Attestation, error)
}

// Handler defines an interface for a struct equipped for receiving block operations.
type Handler interface {
	HandleAttestation(context.Context, proto.Message) error
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

// AttestationPoolForForkchoice returns the attestations that have not been processed by the
// fork choice service. It will not return the attestations which the validator vote has
// already been counted.
func (s *Service) AttestationPoolForForkchoice(ctx context.Context) ([]*ethpb.Attestation, error) {
	s.attestationPoolLock.Lock()
	defer s.attestationPoolLock.Unlock()

	atts := make([]*ethpb.Attestation, 0, len(s.attestationPool))

	for root, ac := range s.attestationPool {
		for i, att := range ac.ToAttestations() {
			if ac.SignaturePairs[i].VoteCounted {
				continue
			}
			if s.recentAttestationBitlist.Contains(root, att.AggregationBits) {
				continue
			}
			atts = append(atts, att)
			ac.SignaturePairs[i].VoteCounted = true
		}
	}

	return atts, nil
}

// AttestationPool returns the attestations that have not seen on the beacon chain,
// the attestations are returned in target epoch ascending order and up to MaxAttestations
// capacity. The attestations returned will be verified against the head state up to requested slot.
// When fails attestation, the attestation will be removed from the pool.
func (s *Service) AttestationPool(ctx context.Context, requestedSlot uint64) ([]*ethpb.Attestation, error) {
	ctx, span := trace.StartSpan(ctx, "operations.AttestationPool")
	defer span.End()

	s.attestationPoolLock.Lock()
	defer s.attestationPoolLock.Unlock()

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

// HandleAttestation processes a received attestation message.
func (s *Service) HandleAttestation(ctx context.Context, message proto.Message) error {
	ctx, span := trace.StartSpan(ctx, "operations.HandleAttestation")
	defer span.End()

	s.attestationPoolLock.Lock()
	defer s.attestationPoolLock.Unlock()

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
