package detection

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "detection")

// Service struct for the detection service of the slasher.
type Service struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	blocksChan            chan *ethpb.SignedBeaconBlock
	attsChan              chan *ethpb.Attestation
	notifier              beaconclient.Notifier
	chainFetcher          beaconclient.ChainFetcher
	historicalDataFetcher beaconclient.HistoricalFetcher
	attesterSlashingsFeed *event.Feed
	proposerSlashingsFeed *event.Feed
}

// Config options for the detection service.
type Config struct {
	Notifier              beaconclient.Notifier
	ChainFetcher          beaconclient.ChainFetcher
	HistoricalDataFetcher beaconclient.HistoricalFetcher
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
		chainFetcher:          cfg.ChainFetcher,
		historicalDataFetcher: cfg.HistoricalDataFetcher,
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

	// We wait for the gRPC beacon client to be ready and the beacon node
	// to be fully synced before proceeding.
	ch := make(chan bool)
	sub := ds.notifier.ClientReadyFeed().Subscribe(ch)
	<-ch
	sub.Unsubscribe()

	// The detection service runs detection on all historical
	// chain data since genesis.
	go ds.detectHistoricalChainData(ds.ctx)

	// We subscribe to incoming blocks from the beacon node via
	// our gRPC client to keep detecting slashable offenses.
	go ds.detectIncomingBlocks(ds.ctx, ds.blocksChan)
	go ds.detectIncomingAttestations(ds.ctx, ds.attsChan)
}

func (ds *Service) detectHistoricalChainData(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "detection.detectHistoricalChainData")
	defer span.End()
	head, err := ds.chainFetcher.ChainHead(ds.ctx)
	if err != nil {
		log.Fatalf("Cannot retrieve chain head: %v", err)
	}
	for i := uint64(0); i < head.HeadEpoch; i++ {
		indexedAtts, err := ds.historicalDataFetcher.RequestHistoricalAttestations(ds.ctx, i /* epoch */)
		if err != nil {
			log.WithError(err).Errorf("Could not fetch attestations for epoch: %d", i)
		}
		log.Infof(
			"Running slashing detection on %d attestations in epoch %d...",
			len(indexedAtts),
			i,
		)
		// TODO(#4836): Run detection function for attester double voting.
		// TODO(#4836): Run detection function for attester surround voting.
	}
	log.Infof("Completed slashing detection on historical chain data up to epoch %d", head.HeadEpoch)
}
