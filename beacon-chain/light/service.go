package light

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	Database            iface.LightClientDatabase
	HeadFetcher         blockchain.HeadFetcher
	FinalizationFetcher blockchain.FinalizationFetcher
	StateNotifier       statefeed.Notifier
	TimeFetcher         blockchain.TimeFetcher
	SyncChecker         sync.Checker
	cancelFunc          context.CancelFunc
	prevHeadData        map[[32]byte]*ethpb.SyncAttestedData
}

func (s *Service) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel
	s.waitForSync(ctx, s.TimeFetcher.GenesisTime())
	go s.listenForNewHead(ctx)
}

func (s *Service) Stop() error {
	s.cancelFunc()
	return nil
}

func (s *Service) Status() error {
	return nil
}

func (s *Service) listenForNewHead(ctx context.Context) {
	stateChan := make(chan *feed.Event, 1)
	sub := s.StateNotifier.StateFeed().Subscribe(stateChan)
	defer sub.Unsubscribe()
	select {
	case ev := <-stateChan:
		if ev.Type == statefeed.NewHead {
			head, err := s.HeadFetcher.HeadBlock(ctx)
			if err != nil {
				log.Error(err)
				return
			}
			st, err := s.HeadFetcher.HeadState(ctx)
			if err != nil {
				log.Error(err)
				return
			}
			if err := s.onHead(st, head.Block()); err != nil {
				log.Error(err)
				return
			}
		} else if ev.Type == statefeed.FinalizedCheckpoint {
			st, err := s.HeadFetcher.HeadState(ctx)
			if err != nil {
				log.Error(err)
				return
			}
			cpt := s.FinalizationFetcher.FinalizedCheckpt()
			if err := s.onFinalized(st, cpt); err != nil {
				log.Error(err)
				return
			}
		}
	case <-sub.Err():
		return
	case <-ctx.Done():
		return
	}
}

func (s *Service) waitForSync(ctx context.Context, genesisTime time.Time) {
	if slots.SinceGenesis(genesisTime) == 0 || !s.SyncChecker.Syncing() {
		return
	}
	slotTicker := slots.NewSlotTicker(genesisTime, params.BeaconConfig().SecondsPerSlot)
	defer slotTicker.Done()
	for {
		select {
		case <-slotTicker.C():
			// If node is still syncing, do not operate.
			if s.SyncChecker.Syncing() {
				continue
			}
			return
		case <-ctx.Done():
			return
		}
	}
}
