// Package attester defines all relevant functionality for an Attester actor
// within a sharded Ethereum blockchain.
package attester

import (
	"github.com/prysmaticlabs/geth-sharding/sharding/database"
	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
	"github.com/prysmaticlabs/geth-sharding/sharding/params"
	log "github.com/sirupsen/logrus"
)

// Attester holds functionality required to run a collation attester
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Attester struct {
	config    *params.Config
	smcClient *mainchain.SMCClient
	p2p       *p2p.Server
	dbService *database.ShardDB
}

// NewAttester creates a new attester instance.
func NewAttester(config *params.Config, smcClient *mainchain.SMCClient, p2p *p2p.Server, dbService *database.ShardDB) (*Attester, error) {
	return &Attester{config, smcClient, p2p, dbService}, nil
}

// Start the main routine for an attester.
func (n *Attester) Start() {
	log.Info("Starting attester service")
	go n.attestCollations()
}

// Stop the main loop for attesting collations.
func (n *Attester) Stop() error {
	log.Info("Stopping attester service")
	return nil
}

// attestCollations checks incoming block headers and determines if
// we are an eligible attester for collations.
func (n *Attester) attestCollations() {

	// TODO: handle this better through goroutines. Right now, these methods
	// are blocking.
	if n.smcClient.DepositFlag() {
		if err := joinAttesterPool(n.smcClient, n.smcClient, n.config); err != nil {
			log.Errorf("Could not fetch current block number: %v", err)
			return
		}
	}

	if err := subscribeBlockHeaders(n.smcClient.ChainReader(), n.smcClient, n.smcClient.Account()); err != nil {
		log.Errorf("Could not fetch current block number: %v", err)
		return
	}
}
