package detection

import (
	"context"
	"errors"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/iface"
	"github.com/prysmaticlabs/prysm/slasher/detection/proposals"
	proposerIface "github.com/prysmaticlabs/prysm/slasher/detection/proposals/iface"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// Status detection statuses type.
type Status int

const (
	// None slasher was not initialised.
	None Status = iota
	// Started service start has been called,
	Started
	// Syncing beacon client is still syncing.
	Syncing
	// HistoricalDetection slasher is replaying all attestations that
	// were included in the canonical chain.
	HistoricalDetection
	// Ready slasher is ready to detect requests.
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
	cfg                *Config
	ctx                context.Context
	cancel             context.CancelFunc
	blocksChan         chan *ethpb.SignedBeaconBlock
	attsChan           chan *ethpb.IndexedAttestation
	minMaxSpanDetector iface.SpanDetector
	proposalsDetector  proposerIface.ProposalsDetector
	status             Status
}

// Config options for the detection service.
type Config struct {
	Notifier              beaconclient.Notifier
	SlasherDB             db.Database
	ChainFetcher          beaconclient.ChainFetcher
	BeaconClient          *beaconclient.Service
	AttesterSlashingsFeed *event.Feed
	ProposerSlashingsFeed *event.Feed
	HistoricalDetection   bool
}

// NewService instantiation.
func NewService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		cfg:                cfg,
		ctx:                ctx,
		cancel:             cancel,
		blocksChan:         make(chan *ethpb.SignedBeaconBlock, 1),
		attsChan:           make(chan *ethpb.IndexedAttestation, 1),
		minMaxSpanDetector: attestations.NewSpanDetector(cfg.SlasherDB),
		proposalsDetector:  proposals.NewProposeDetector(cfg.SlasherDB),
		status:             None,
	}
}

// Stop the notifier service.
func (s *Service) Stop() error {
	s.cancel()
	log.Info("Stopping service")
	return nil
}

// Status returns an error if detection service is not ready yet.
func (s *Service) Status() error {
	if s.status == Ready {
		return nil
	}
	return errors.New(s.status.String())
}

// Start the detection service runtime.
func (s *Service) Start() {
	// We wait for the gRPC beacon client to be ready and the beacon node
	// to be fully synced before proceeding.
	s.status = Started
	ch := make(chan bool)
	sub := s.cfg.Notifier.ClientReadyFeed().Subscribe(ch)
	s.status = Syncing
	<-ch
	sub.Unsubscribe()

	if s.cfg.HistoricalDetection {
		// The detection service runs detection on all historical
		// chain data since genesis.
		s.status = HistoricalDetection
		s.detectHistoricalChainData(s.ctx)
	}
	s.status = Ready
	// We listen to a stream of blocks and attestations from the beacon node.
	go s.cfg.BeaconClient.ReceiveBlocks(s.ctx)
	go s.cfg.BeaconClient.ReceiveAttestations(s.ctx)
	// We subscribe to incoming blocks from the beacon node via
	// our gRPC client to keep detecting slashable offenses.
	go s.detectIncomingBlocks(s.ctx, s.blocksChan)
	go s.detectIncomingAttestations(s.ctx, s.attsChan)
}

func (s *Service) detectHistoricalChainData(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "detection.detectHistoricalChainData")
	defer span.End()
	// We fetch both the latest persisted chain head in our DB as well
	// as the current chain head from the beacon node via gRPC.
	latestStoredHead, err := s.cfg.SlasherDB.ChainHead(ctx)
	if err != nil {
		log.WithError(err).Error("Could not retrieve chain head from DB")
		return
	}
	currentChainHead, err := s.cfg.ChainFetcher.ChainHead(ctx)
	if err != nil {
		log.WithError(err).Error("Cannot retrieve chain head from beacon node")
		return
	}
	var latestStoredEpoch types.Epoch
	if latestStoredHead != nil {
		latestStoredEpoch = latestStoredHead.HeadEpoch
	}
	log.Infof("Performing historical detection from epoch %d to %d", latestStoredEpoch, currentChainHead.HeadEpoch)

	// We retrieve historical chain data from the last persisted chain head in the
	// slasher DB up to the current beacon node's head epoch we retrieved via gRPC.
	// If no data was persisted from previous sessions, we request data starting from
	// the genesis epoch.
	var storedEpoch types.Epoch
	for epoch := latestStoredEpoch; epoch < currentChainHead.HeadEpoch; epoch++ {
		if ctx.Err() != nil {
			log.WithError(err).Errorf("Could not fetch attestations for epoch: %d", epoch)
			return
		}
		indexedAtts, err := s.cfg.BeaconClient.RequestHistoricalAttestations(ctx, epoch)
		if err != nil {
			log.WithError(err).Errorf("Could not fetch attestations for epoch: %d", epoch)
			return
		}
		if err := s.cfg.SlasherDB.SaveIndexedAttestations(ctx, indexedAtts); err != nil {
			log.WithError(err).Error("could not save indexed attestations")
			return
		}

		for _, att := range indexedAtts {
			if ctx.Err() == context.Canceled {
				log.WithError(ctx.Err()).Error("context has been canceled, ending detection")
				return
			}
			slashings, err := s.DetectAttesterSlashings(ctx, att)
			if err != nil {
				log.WithError(err).Error("Could not detect attester slashings")
				continue
			}
			if len(slashings) < 1 {
				if err := s.minMaxSpanDetector.UpdateSpans(ctx, att); err != nil {
					log.WithError(err).Error("Could not update spans")
				}
			}
			s.submitAttesterSlashings(ctx, slashings)

			if err := s.UpdateHighestAttestation(ctx, att); err != nil {
				log.WithError(err).Errorf("Could not update highest attestation")
			}
		}
		latestStoredHead = &ethpb.ChainHead{HeadEpoch: epoch}
		if err := s.cfg.SlasherDB.SaveChainHead(ctx, latestStoredHead); err != nil {
			log.WithError(err).Error("Could not persist chain head to disk")
		}
		storedEpoch = epoch
		s.cfg.SlasherDB.RemoveOldestFromCache(ctx)
		if epoch == currentChainHead.HeadEpoch-1 {
			currentChainHead, err = s.cfg.ChainFetcher.ChainHead(ctx)
			if err != nil {
				log.WithError(err).Error("Cannot retrieve chain head from beacon node")
				return
			}
			if epoch != currentChainHead.HeadEpoch-1 {
				log.Infof("Continuing historical detection from epoch %d to %d", epoch, currentChainHead.HeadEpoch)
			}
		}
	}
	log.Infof("Completed slashing detection on historical chain data up to epoch %d", storedEpoch)
}

func (s *Service) submitAttesterSlashings(ctx context.Context, slashings []*ethpb.AttesterSlashing) {
	ctx, span := trace.StartSpan(ctx, "detection.submitAttesterSlashings")
	defer span.End()
	for i := 0; i < len(slashings); i++ {
		s.cfg.AttesterSlashingsFeed.Send(slashings[i])
	}
}

func (s *Service) submitProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) {
	ctx, span := trace.StartSpan(ctx, "detection.submitProposerSlashing")
	defer span.End()
	if slashing != nil && slashing.Header_1 != nil && slashing.Header_2 != nil {
		log.WithFields(logrus.Fields{
			"header1Slot":        slashing.Header_1.Header.Slot,
			"header2Slot":        slashing.Header_2.Header.Slot,
			"proposerIdxHeader1": slashing.Header_1.Header.ProposerIndex,
			"proposerIdxHeader2": slashing.Header_2.Header.ProposerIndex,
		}).Info("Found a proposer slashing! Submitting to beacon node")
		s.cfg.ProposerSlashingsFeed.Send(slashing)
	}
}
