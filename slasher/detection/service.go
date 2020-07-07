package detection

import (
	"context"
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/iface"
	"github.com/prysmaticlabs/prysm/slasher/detection/proposals"
	proposerIface "github.com/prysmaticlabs/prysm/slasher/detection/proposals/iface"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "detection")

// Status detection statuses type.
type Status int

const (
	None Status = iota
	Started
	Syncing
	HistoricalDetection
	Ready
)

// String returns the string value of the status
func (s Status) String() string {
	strings := [...]string{"None", "Started", "Syncing", "HistoricalDetection", "Ready"}

	// prevent panicking in case of status is out-of-range
	if s < None || s > Ready {
		return "Unknown"
	}

	return strings[s]
}

// Service struct for the detection service of the slasher.
type Service struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	slasherDB             db.Database
	blocksChan            chan *ethpb.SignedBeaconBlock
	attsChan              chan *ethpb.IndexedAttestation
	notifier              beaconclient.Notifier
	chainFetcher          beaconclient.ChainFetcher
	beaconClient          *beaconclient.Service
	attesterSlashingsFeed *event.Feed
	proposerSlashingsFeed *event.Feed
	minMaxSpanDetector    iface.SpanDetector
	proposalsDetector     proposerIface.ProposalsDetector
	status                Status
}

// Config options for the detection service.
type Config struct {
	Notifier              beaconclient.Notifier
	SlasherDB             db.Database
	ChainFetcher          beaconclient.ChainFetcher
	BeaconClient          *beaconclient.Service
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
		slasherDB:             cfg.SlasherDB,
		beaconClient:          cfg.BeaconClient,
		blocksChan:            make(chan *ethpb.SignedBeaconBlock, 1),
		attsChan:              make(chan *ethpb.IndexedAttestation, 1),
		attesterSlashingsFeed: cfg.AttesterSlashingsFeed,
		proposerSlashingsFeed: cfg.ProposerSlashingsFeed,
		minMaxSpanDetector:    attestations.NewSpanDetector(cfg.SlasherDB),
		proposalsDetector:     proposals.NewProposeDetector(cfg.SlasherDB),
		status:                None,
	}
}

// Stop the notifier service.
func (ds *Service) Stop() error {
	ds.cancel()
	log.Info("Stopping service")
	return nil
}

// Status returns an error if detection service is not ready yet.
func (ds *Service) Status() error {
	if ds.status == Ready {
		return nil
	}
	return errors.New(ds.status.String())
}

// Start the detection service runtime.
func (ds *Service) Start() {
	// We wait for the gRPC beacon client to be ready and the beacon node
	// to be fully synced before proceeding.
	ds.status = Started
	ch := make(chan bool)
	sub := ds.notifier.ClientReadyFeed().Subscribe(ch)
	ds.status = Syncing
	<-ch
	sub.Unsubscribe()

	if featureconfig.Get().EnableHistoricalDetection {
		// The detection service runs detection on all historical
		// chain data since genesis.
		ds.status = HistoricalDetection
		ds.detectHistoricalChainData(ds.ctx)
	}
	ds.status = Ready
	// We listen to a stream of blocks and attestations from the beacon node.
	go ds.beaconClient.ReceiveBlocks(ds.ctx)
	go ds.beaconClient.ReceiveAttestations(ds.ctx)
	// We subscribe to incoming blocks from the beacon node via
	// our gRPC client to keep detecting slashable offenses.
	go ds.detectIncomingBlocks(ds.ctx, ds.blocksChan)
	go ds.detectIncomingAttestations(ds.ctx, ds.attsChan)
}

func (ds *Service) detectHistoricalChainData(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "detection.detectHistoricalChainData")
	defer span.End()
	// We fetch both the latest persisted chain head in our DB as well
	// as the current chain head from the beacon node via gRPC.
	latestStoredHead, err := ds.slasherDB.ChainHead(ctx)
	if err != nil {
		log.WithError(err).Error("Could not retrieve chain head from DB")
		return
	}
	currentChainHead, err := ds.chainFetcher.ChainHead(ctx)
	if err != nil {
		log.WithError(err).Error("Cannot retrieve chain head from beacon node")
		return
	}
	var latestStoredEpoch uint64
	if latestStoredHead != nil {
		latestStoredEpoch = latestStoredHead.HeadEpoch
	}
	log.Infof("Performing historical detection from epoch %d to %d", latestStoredEpoch, currentChainHead.HeadEpoch)

	// We retrieve historical chain data from the last persisted chain head in the
	// slasher DB up to the current beacon node's head epoch we retrieved via gRPC.
	// If no data was persisted from previous sessions, we request data starting from
	// the genesis epoch.
	var storedEpoch uint64
	for epoch := latestStoredEpoch; epoch < currentChainHead.HeadEpoch; epoch++ {
		if ctx.Err() != nil {
			log.WithError(err).Errorf("Could not fetch attestations for epoch: %d", epoch)
			return
		}
		indexedAtts, err := ds.beaconClient.RequestHistoricalAttestations(ctx, epoch)
		if err != nil {
			log.WithError(err).Errorf("Could not fetch attestations for epoch: %d", epoch)
			continue
		}
		if err := ds.slasherDB.SaveIndexedAttestations(ctx, indexedAtts); err != nil {
			log.WithError(err).Error("could not save indexed attestations")
			continue
		}

		for _, att := range indexedAtts {
			if ctx.Err() == context.Canceled {
				log.WithError(ctx.Err()).Error("context has been canceled, ending detection")
				return
			}
			slashings, err := ds.DetectAttesterSlashings(ctx, att)
			if err != nil {
				log.WithError(err).Error("Could not detect attester slashings")
				continue
			}
			if len(slashings) < 1 {
				if err := ds.minMaxSpanDetector.UpdateSpans(ctx, att); err != nil {
					log.WithError(err).Error("Could not update spans")
				}
			}
			ds.submitAttesterSlashings(ctx, slashings)
		}
		latestStoredHead = &ethpb.ChainHead{HeadEpoch: epoch}
		if err := ds.slasherDB.SaveChainHead(ctx, latestStoredHead); err != nil {
			log.WithError(err).Error("Could not persist chain head to disk")
		}
		storedEpoch = epoch
		ds.slasherDB.RemoveOldestFromCache(ctx)
		if epoch == currentChainHead.HeadEpoch-1 {
			currentChainHead, err = ds.chainFetcher.ChainHead(ctx)
			if err != nil {
				log.WithError(err).Error("Cannot retrieve chain head from beacon node")
				continue
			}
			if epoch != currentChainHead.HeadEpoch-1 {
				log.Infof("Continuing historical detection from epoch %d to %d", epoch, currentChainHead.HeadEpoch)
			}
		}
	}
	log.Infof("Completed slashing detection on historical chain data up to epoch %d", storedEpoch)
}

func (ds *Service) submitAttesterSlashings(ctx context.Context, slashings []*ethpb.AttesterSlashing) {
	ctx, span := trace.StartSpan(ctx, "detection.submitAttesterSlashings")
	defer span.End()
	for i := 0; i < len(slashings); i++ {
		ds.attesterSlashingsFeed.Send(slashings[i])
	}
}

func (ds *Service) submitProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) {
	ctx, span := trace.StartSpan(ctx, "detection.submitProposerSlashing")
	defer span.End()
	if slashing != nil && slashing.Header_1 != nil && slashing.Header_2 != nil {
		log.WithFields(logrus.Fields{
			"header1Slot":        slashing.Header_1.Header.Slot,
			"header2Slot":        slashing.Header_2.Header.Slot,
			"proposerIdxHeader1": slashing.Header_1.Header.ProposerIndex,
			"proposerIdxHeader2": slashing.Header_2.Header.ProposerIndex,
		}).Info("Found a proposer slashing! Submitting to beacon node")
		ds.proposerSlashingsFeed.Send(slashing)
	}
}
