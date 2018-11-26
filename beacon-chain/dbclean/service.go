// Package
package dbclean

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/event"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "dbcleaner")

type chainService interface {
	CanonicalCrystallizedStateFeed() *event.Feed
}

type DBCleanService struct {
	ctx                context.Context
	cancel             context.CancelFunc
	beaconDB           *db.BeaconDB
	chainService       chainService
	canonicalStateChan chan *types.CrystallizedState
}

type Config struct {
	SubscriptionBuf int
	BeaconDB        *db.BeaconDB
	ChainService    chainService
}

func NewDBCleanService(ctx context.Context, cfg *Config) *DBCleanService {
	ctx, cancel := context.WithCancel(ctx)
	return &DBCleanService{
		ctx:                ctx,
		cancel:             cancel,
		beaconDB:           cfg.BeaconDB,
		chainService:       cfg.ChainService,
		canonicalStateChan: make(chan *types.CrystallizedState, cfg.SubscriptionBuf),
	}
}

func (d *DBCleanService) Start() {
	log.Infoln("Starting DBCleanService")
	go d.cleanBlockVoteCache(d.canonicalStateChan)
}

func (d *DBCleanService) Stop() error {
	defer d.cancel()

	log.Info("Stopping service")
	return nil
}

func (d *DBCleanService) cleanBlockVoteCache(newCState <-chan *types.CrystallizedState) {
	cStateSub := d.chainService.CanonicalCrystallizedStateFeed().Subscribe(d.canonicalStateChan)
	defer cStateSub.Unsubscribe()

	for {
		select {
		case <-d.ctx.Done():
			log.Debug("DB Clean service context closed, exiting goroutine")
			return
		case cState := <-d.canonicalStateChan:
			log.Infoln("DBCleanService receive new crystallized state")
			log.Infoln("%v", cState)
		}
	}
}
