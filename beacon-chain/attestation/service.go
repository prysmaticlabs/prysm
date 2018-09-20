// Package attestation defines the life-cycle and status of single and aggregated attestation.
package attestation

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/bazel-go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "attestation")

// AttestationService represents a service that handles the internal
// logic of managing aggregated attestation.
type AttestationService struct {
	ctx                      context.Context
	cancel                   context.CancelFunc
	core                     *AttestationHandler
	broadcastFeed            *event.Feed
	broadcastChan            chan *types.Attestation
	receiveFeed              *event.Feed
	receiveChan              chan *types.Attestation
	processedAttestationFeed *event.Feed
}

// Config options for the service.
type Config struct {
	handler                 *AttestationHandler
	receiveAttestationBuf   int
	broadcastAttestationBuf int
}

// NewAttestationService instantiates a new service instance that will
// be registered into a running beacon node.
func NewAttestationService(ctx context.Context, cfg *Config) *AttestationService {
	ctx, cancel := context.WithCancel(ctx)
	return &AttestationService{
		ctx:                      ctx,
		cancel:                   cancel,
		core:                     cfg.handler,
		broadcastFeed:            new(event.Feed),
		broadcastChan:            make(chan *types.Attestation, cfg.broadcastAttestationBuf),
		receiveFeed:              new(event.Feed),
		receiveChan:              make(chan *types.Attestation, cfg.receiveAttestationBuf),
		processedAttestationFeed: new(event.Feed),
	}
}

// Start an attestation service's main event loop.
func (a *AttestationService) Start() {
	log.Info("Starting service")
}

// Stop the Attestation service's main event loop and associated goroutines.
func (a *AttestationService) Stop() error {
	defer a.cancel()
	log.Info("Stopping service")
	return nil
}

// IncomingAttestationFeed returns a feed that any service can send incoming p2p attestations into.
// The chain service will subscribe to this feed in order to relay incoming attestations.
func (a *AttestationService) IncomingAttestationFeed() *event.Feed {
	return a.IncomingAttestationFeed()
}

// ProcessedAttestationFeed returns a feed that will be used to stream attestations that have been
// processed by the beacon node to its rpc clients.
func (a *AttestationService) ProcessedAttestationFeed() *event.Feed {
	return a.processedAttestationFeed
}

// ContainsAttestation checks if an attestation has already been aggregated.
func (a *AttestationService) ContainsAttestation(bitfield []byte, h [32]byte) (bool, error) {
	attestation, err := a.core.getAttestation(h)
	if err != nil {
		return false, fmt.Errorf("could not get attestation from DB: %v", err)
	}
	savedAttestationBitfield := attestation.AttesterBitfield()
	for i := 0; i < len(bitfield); i++ {
		if bitfield[i]&savedAttestationBitfield[i] != 0 {
			return true, nil
		}
	}
	return false, nil
}

func (a *AttestationService) attestationProcessing() {
	broadcastSub := a.broadcastFeed.Subscribe(a.broadcastChan)
	receiveSub := a.receiveFeed.Subscribe(a.receiveChan)

	defer broadcastSub.Unsubscribe()
	defer receiveSub.Unsubscribe()
	for {
		select {
		case <-a.ctx.Done():
			log.Debug("Attestation service context closed, exiting goroutine")
			return
			// Listen for a newly received incoming attestation from the sync service.
		case attestation := <-a.receiveChan:
			h, err := attestation.Hash()
			if err != nil {
				log.Debugf("Could not hash incoming attestation: %v", err)
			}
			if err := a.core.saveAttestation(attestation); err != nil {
				log.Errorf("Could not save attestation: %v", err)
				continue
			}

			a.processedAttestationFeed.Send(attestation.Proto)
			log.Info("Relaying attestation 0x%v to proposers through grpc", h)
		}
	}
}
