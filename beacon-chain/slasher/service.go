package slasher

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
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
	StateGen                stategen.StateManager
	SlashingPoolInserter    slashings.PoolManager
	HeadStateFetcher        blockchain.HeadFetcher
}

// Service for running slasher mode in a beacon node.
type Service struct {
	params     *Parameters
	serviceCfg *ServiceConfig
}
