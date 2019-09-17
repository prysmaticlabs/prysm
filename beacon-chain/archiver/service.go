package archiver

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "archiver")

type Service struct {
	ctx             context.Context
	beaconDB        db.Database
	newHeadNotifier blockchain.NewHeadNotifier
}

type Config struct {
	BeaconDB        db.Database
	NewHeadNotifier blockchain.NewHeadNotifier
}

func NewArchiverService(ctx context.Context, cfg *Config) *Service {
	return &Service{
		ctx:             ctx,
		beaconDB:        cfg.BeaconDB,
		newHeadNotifier: cfg.NewHeadNotifier,
	}
}

func (s *Service) Start() {
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) Status() error {
	return nil
}

func (s *Service) run() {
	headRootCh := make(chan [32]byte, 1)
	sub := s.newHeadNotifier.HeadUpdatedFeed().Subscribe(headRootCh)
	defer sub.Unsubscribe()
	for {
		select {
		case h := <-headRootCh:
			log.WithField("headRoot", fmt.Sprintf("%#x", h)).Info("New chain head event")
		case err := <-sub.Err():
			log.WithError(err).Error("Subscription to new chain head notifier failed")
		}
	}
}
