package light

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	cancelFunc          context.CancelFunc
	Database            iface.LightClientDatabase
	HeadFetcher         blockchain.HeadFetcher
	FinalizationFetcher blockchain.FinalizationFetcher
	StateNotifier       statefeed.Notifier
	prevHeadData        map[[32]byte]*ethpb.SyncAttestedData
}

func (s *Service) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel
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
