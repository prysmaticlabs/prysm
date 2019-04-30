// Package attestation defines the life-cycle and status of single and aggregated attestation.
package attestation

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	handler "github.com/prysmaticlabs/prysm/shared/messagehandler"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "attestation")
var committeeCache = cache.NewCommitteesCache()

// TargetHandler provides an interface for fetching latest attestation targets
// and updating attestations in batches.
type TargetHandler interface {
	LatestAttestationTarget(state *pb.BeaconState, validatorIndex uint64) (*pb.AttestationTarget, error)
	BatchUpdateLatestAttestation(ctx context.Context, atts []*pb.Attestation) error
}

type attestationStore struct {
	sync.RWMutex
	m map[[48]byte]*pb.Attestation
}

// Service represents a service that handles the internal
// logic of managing single and aggregated attestation.
type Service struct {
	ctx          context.Context
	cancel       context.CancelFunc
	beaconDB     *db.BeaconDB
	incomingFeed *event.Feed
	incomingChan chan *pb.Attestation
	// store is the mapping of individual
	// validator's public key to it's latest attestation.
	store attestationStore
}

// Config options for the service.
type Config struct {
	BeaconDB *db.BeaconDB
}

// NewAttestationService instantiates a new service instance that will
// be registered into a running beacon node.
func NewAttestationService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:          ctx,
		cancel:       cancel,
		beaconDB:     cfg.BeaconDB,
		incomingFeed: new(event.Feed),
		incomingChan: make(chan *pb.Attestation, params.BeaconConfig().DefaultBufferSize),
		store:        attestationStore{m: make(map[[48]byte]*pb.Attestation)},
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

// LatestAttestationTarget returns the target block that the validator index attested to,
// the highest slotNumber attestation in attestation pool gets returned.
//
// Spec pseudocode definition:
//	Let `get_latest_attestation_target(store: Store, validator_index: ValidatorIndex) ->
//		BeaconBlock` be the target block in the attestation
//		`get_latest_attestation(store, validator_index)`.
func (a *Service) LatestAttestationTarget(beaconState *pb.BeaconState, index uint64) (*pb.AttestationTarget, error) {
	if index >= uint64(len(beaconState.ValidatorRegistry)) {
		return nil, fmt.Errorf("invalid validator index %d", index)
	}
	validator := beaconState.ValidatorRegistry[index]

	pubKey := bytesutil.ToBytes48(validator.Pubkey)
	a.store.RLock()
	defer a.store.RUnlock()
	if _, exists := a.store.m[pubKey]; !exists {
		return nil, nil
	}

	attestation := a.store.m[pubKey]
	if attestation == nil {
		return nil, nil
	}
	targetRoot := bytesutil.ToBytes32(attestation.Data.BeaconBlockRootHash32)
	if !a.beaconDB.HasBlock(targetRoot) {
		return nil, nil
	}

	return a.beaconDB.AttestationTarget(targetRoot)
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
		case attestations := <-a.incomingChan:
			handler.SafelyHandleMessage(a.ctx, a.handleAttestation, attestations)
		}
	}
}

func (a *Service) handleAttestation(ctx context.Context, msg proto.Message) error {
	attestation := msg.(*pb.Attestation)
	if err := a.UpdateLatestAttestation(ctx, attestation); err != nil {
		return fmt.Errorf("could not update attestation pool: %v", err)
	}
	return nil
}

// UpdateLatestAttestation inputs an new attestation and checks whether
// the attesters who submitted this attestation with the higher slot number
// have been noted in the attestation pool. If not, it updates the
// attestation pool with attester's public key to attestation.
func (a *Service) UpdateLatestAttestation(ctx context.Context, attestation *pb.Attestation) error {
	totalAttestationSeen.Inc()

	// Potential improvement, instead of getting the state,
	// we could get a mapping of validator index to public key.
	beaconState, err := a.beaconDB.HeadState(ctx)
	if err != nil {
		return err
	}
	head, err := a.beaconDB.ChainHead()
	if err != nil {
		return err
	}
	headRoot, err := hashutil.HashBeaconBlock(head)
	if err != nil {
		return err
	}
	return a.updateAttestation(ctx, headRoot, beaconState, attestation)
}

// BatchUpdateLatestAttestation updates multiple attestations and adds them into the attestation store
// if they are valid.
func (a *Service) BatchUpdateLatestAttestation(ctx context.Context, attestations []*pb.Attestation) error {

	if attestations == nil {
		return nil
	}
	// Potential improvement, instead of getting the state,
	// we could get a mapping of validator index to public key.
	beaconState, err := a.beaconDB.HeadState(ctx)
	if err != nil {
		return err
	}
	head, err := a.beaconDB.ChainHead()
	if err != nil {
		return err
	}
	headRoot, err := hashutil.HashBeaconBlock(head)
	if err != nil {
		return err
	}

	attestations = a.sortAttestations(attestations)

	for _, attestation := range attestations {
		if err := a.updateAttestation(ctx, headRoot, beaconState, attestation); err != nil {
			return err
		}
	}
	return nil
}

// InsertAttestationIntoStore locks the store, inserts the attestation, then
// unlocks the store again. This method may be used by external services
// in testing to populate the attestation store.
func (a *Service) InsertAttestationIntoStore(pubkey [48]byte, att *pb.Attestation) {
	a.store.Lock()
	defer a.store.Unlock()
	a.store.m[pubkey] = att
}

func (a *Service) updateAttestation(ctx context.Context, headRoot [32]byte, beaconState *pb.BeaconState,
	attestation *pb.Attestation) error {
	totalAttestationSeen.Inc()

	slot := attestation.Data.Slot
	var committee []uint64
	var cachedCommittees *cache.CommitteesInSlot
	var err error

	for beaconState.Slot < slot {
		beaconState, err = state.ExecuteStateTransition(
			ctx, beaconState, nil /* block */, headRoot, &state.TransitionConfig{},
		)
		if err != nil {
			return fmt.Errorf("could not execute head transition: %v", err)
		}
	}

	cachedCommittees, err = committeeCache.CommitteesInfoBySlot(slot)
	if err != nil {
		return err
	}
	if cachedCommittees == nil {
		crosslinkCommittees, err := helpers.CrosslinkCommitteesAtSlot(beaconState, slot, false /* registryChange */)
		if err != nil {
			return err
		}
		cachedCommittees = helpers.ToCommitteeCache(slot, crosslinkCommittees)
		if err := committeeCache.AddCommittees(cachedCommittees); err != nil {
			return err
		}
	}

	// Find committee for shard.
	for _, v := range cachedCommittees.Committees {
		if v.Shard == attestation.Data.Shard {
			committee = v.Committee
			break
		}
	}

	log.WithFields(logrus.Fields{
		"attestationSlot":    attestation.Data.Slot - params.BeaconConfig().GenesisSlot,
		"attestationShard":   attestation.Data.Shard,
		"committeesShard":    cachedCommittees.Committees[0].Shard,
		"committeesList":     cachedCommittees.Committees[0].Committee,
		"lengthOfCommittees": len(cachedCommittees.Committees),
	}).Debug("Updating latest attestation")

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

		if i >= len(committee) {
			log.Errorf("Bitfield points to an invalid index in the committee: bitfield %08b", bitfield)
			continue
		}

		if int(committee[i]) >= len(beaconState.ValidatorRegistry) {
			log.Errorf("Index doesn't exist in validator registry: index %d", committee[i])
		}

		// If the attestation came from this attester. We use the slot committee to find the
		// validator's actual index.
		pubkey := bytesutil.ToBytes48(beaconState.ValidatorRegistry[committee[i]].Pubkey)
		newAttestationSlot := attestation.Data.Slot
		currentAttestationSlot := uint64(0)
		a.store.Lock()
		defer a.store.Unlock()
		if _, exists := a.store.m[pubkey]; exists {
			currentAttestationSlot = a.store.m[pubkey].Data.Slot
		}
		// If the attestation is newer than this attester's one in pool.
		if newAttestationSlot > currentAttestationSlot {
			a.store.m[pubkey] = attestation

			log.WithFields(
				logrus.Fields{
					"attestationSlot": attestation.Data.Slot - params.BeaconConfig().GenesisSlot,
					"justifiedEpoch":  attestation.Data.JustifiedEpoch - params.BeaconConfig().GenesisEpoch,
				},
			).Debug("Attestation store updated")

			blockRoot := bytesutil.ToBytes32(attestation.Data.BeaconBlockRootHash32)
			votedBlock, err := a.beaconDB.Block(blockRoot)
			if err != nil {
				return err
			}
			reportVoteMetrics(committee[i], votedBlock)
		}
	}
	return nil
}

// sortAttestations sorts attestations by their slot number in ascending order.
func (a *Service) sortAttestations(attestations []*pb.Attestation) []*pb.Attestation {
	sort.SliceStable(attestations, func(i, j int) bool {
		return attestations[i].Data.Slot < attestations[j].Data.Slot
	})

	return attestations
}
