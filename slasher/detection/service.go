package detection

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "detection")

// Service struct for the detection service of the slasher.
type Service struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	blocksChan            chan *ethpb.SignedBeaconBlock
	attsChan              chan *ethpb.Attestation
	notifier              beaconclient.Notifier
	attesterSlashingsFeed *event.Feed
	proposerSlashingsFeed *event.Feed
}

// Config options for the detection service.
type Config struct {
	Notifier              beaconclient.Notifier
	AttesterSlashingsFeed *event.Feed
	ProposerSlashingsFeed *event.Feed
}

// NewDetectionService instantiation.
func NewDetectionService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                   ctx,
		cancel:                cancel,
		notifier:              cfg.Notifier,
		blocksChan:            make(chan *ethpb.SignedBeaconBlock, 1),
		attsChan:              make(chan *ethpb.Attestation, 1),
		attesterSlashingsFeed: cfg.AttesterSlashingsFeed,
		proposerSlashingsFeed: cfg.ProposerSlashingsFeed,
	}
}

// Stop the notifier service.
func (ds *Service) Stop() error {
	ds.cancel()
	log.Info("Stopping service")
	return nil
}

// Status returns an error if there exists an error in
// the notifier service.
func (ds *Service) Status() error {
	return nil
}

// Start the detection service runtime.
func (ds *Service) Start() {
	go ds.detectIncomingBlocks(ds.ctx, ds.blocksChan)
	go ds.detectIncomingAttestations(ds.ctx, ds.attsChan)
}
