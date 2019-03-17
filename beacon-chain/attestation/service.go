// Package attestation defines the life-cycle and status of single and aggregated attestation.
package attestation

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"

	handler "github.com/prysmaticlabs/prysm/shared/messagehandler"

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
	// store is the mapping of individual
	// validator's public key to it's latest attestation.
	Store map[[48]byte]*pb.Attestation
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
		Store:         make(map[[48]byte]*pb.Attestation),
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

// LatestAttestation returns the latest attestation from validator index, the highest
// slotNumber attestation from the attestation pool gets returned.
//
// Spec pseudocode definition:
//	Let `get_latest_attestation(store: Store, validator_index: ValidatorIndex) ->
//		Attestation` be the attestation with the highest slot number in `store`
//		from the validator with the given `validator_index`
func (a *Service) LatestAttestation(ctx context.Context, index uint64) (*pb.Attestation, error) {
	state, err := a.beaconDB.State(ctx)
	if err != nil {
		return nil, err
	}

	// return error if it's an invalid validator index.
	if index >= uint64(len(state.ValidatorRegistry)) {
		return nil, fmt.Errorf("invalid validator index %d", index)
	}
	pubKey := bytesutil.ToBytes48(state.ValidatorRegistry[index].Pubkey)

	// return error if validator has no attestation.
	if _, exists := a.Store[pubKey]; !exists {
		return nil, fmt.Errorf("validator index %d does not have an attestation", index)
	}

	return a.Store[pubKey], nil
}

// LatestAttestationTarget returns the target block the validator index attested to,
// the highest slotNumber attestation in attestation pool gets returned.
//
// Spec pseudocode definition:
//	Let `get_latest_attestation_target(store: Store, validator_index: ValidatorIndex) ->
//		BeaconBlock` be the target block in the attestation
//		`get_latest_attestation(store, validator_index)`.
func (a *Service) LatestAttestationTarget(ctx context.Context, index uint64) (*pb.BeaconBlock, error) {
	attestation, err := a.LatestAttestation(ctx, index)
	if err != nil {
		return nil, fmt.Errorf("could not get attestation: %v", err)
	}
	targetBlockHash := bytesutil.ToBytes32(attestation.Data.BeaconBlockRootHash32)
	targetBlock, err := a.beaconDB.Block(targetBlockHash)
	if err != nil {
		return nil, fmt.Errorf("could not get target block: %v", err)
	}
	return targetBlock, nil
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
			handler.SafelyHandleMessage(a.ctx, a.handleAttestation, attestation)
		}
	}
}

func (a *Service) handleAttestation(ctx context.Context, msg proto.Message) error {
	attestation := msg.(*pb.Attestation)
	enc, err := proto.Marshal(attestation)
	if err != nil {
		return fmt.Errorf("could not marshal incoming attestation to bytes: %v", err)
	}
	h := hashutil.Hash(enc)

	if err := a.updateLatestAttestation(ctx, attestation); err != nil {
		return fmt.Errorf("could not update attestation pool: %v", err)
	}
	log.Infof("Updated attestation pool for attestation %#x", h)
	return nil
}

// updateLatestAttestation inputs an new attestation and checks whether
// the attesters who submitted this attestation with the higher slot number
// have been noted in the attestation pool. If not, it updates the
// attestation pool with attester's public key to attestation.
func (a *Service) updateLatestAttestation(ctx context.Context, attestation *pb.Attestation) error {
	// Potential improvement, instead of getting the state,
	// we could get a mapping of validator index to public key.
	state, err := a.beaconDB.State(ctx)
	if err != nil {
		return err
	}

	var committee []uint64
	// We find the crosslink committee for the shard and slot by the attestation.
	committees, err := helpers.CrosslinkCommitteesAtSlot(state, attestation.Data.Slot, false /* registryChange */)
	if err != nil {
		return err
	}

	// Find committee for shard.
	for _, v := range committees {
		if v.Shard == attestation.Data.Shard {
			committee = v.Committee
			break
		}
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

		// If the attestation came from this attester. We use the slot committee to find the
		// validator's actual index.
		pubkey := bytesutil.ToBytes48(state.ValidatorRegistry[committee[i]].Pubkey)
		newAttestationSlot := attestation.Data.Slot
		currentAttestationSlot := uint64(0)
		if _, exists := a.Store[pubkey]; exists {
			currentAttestationSlot = a.Store[pubkey].Data.Slot
		}
		// If the attestation is newer than this attester's one in pool.
		if newAttestationSlot > currentAttestationSlot {
			a.Store[pubkey] = attestation
		}
	}
	return nil
}
