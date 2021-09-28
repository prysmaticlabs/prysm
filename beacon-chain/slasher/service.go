package slasher

import (
	"time"

	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "slasher")

// ServiceConfig contains service dependencies for slasher.
type ServiceConfig struct {
	Database                db.SlasherDatabase
	AttestationStateFetcher blockchain.AttestationStateFetcher
	IndexedAttestationsFeed *event.Feed
	BeaconBlockHeadersFeed  *event.Feed
	StateGen                stategen.StateManager
	SlashingPoolInserter    slashings.PoolManager
	StateNotifier           statefeed.Notifier
	HeadStateFetcher        blockchain.HeadFetcher
}

// Service for running slasher mode in a beacon node.
type Service struct {
	params      *Parameters
	serviceCfg  *ServiceConfig
	blksQueue   *blocksQueue
	attsQueue   *attestationsQueue
	genesisTime time.Time
}
