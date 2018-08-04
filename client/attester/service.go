// Package attester defines all relevant functionality for a Attester actor
// within a sharded Ethereum blockchain.
package attester

import (
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "attester")

// Attester holds functionality required to run a collation attester
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Attester struct {
	config    *params.Config
	smcClient *mainchain.SMCClient
	p2p       *p2p.Server
	dbService *database.DB
}

// NewAttester creates a new attester instance.
func NewAttester(config *params.Config, smcClient *mainchain.SMCClient, p2p *p2p.Server, dbService *database.DB) (*Attester, error) {
	return &Attester{
		config:    config,
		smcClient: smcClient,
		p2p:       p2p,
		dbService: dbService,
	}, nil
}

// Start the main routine for a attester.
func (n *Attester) Start() {
	log.Info("Starting attester service")
	go n.notarizeCollations()
}

// Stop the main loop for notarizing collations.
func (n *Attester) Stop() error {
	log.Info("Stopping attester service")
	return nil
}

// notarizeCollations checks incoming block headers and determines if
// we are an eligible attester for collations.
func (n *Attester) notarizeCollations() {

	// TODO: handle this better through goroutines. Right now, these methods
	// are blocking.
	if n.smcClient.DepositFlag() {
		if err := joinAttesterPool(n.smcClient, n.smcClient); err != nil {
			log.Errorf("Could not fetch current block number: %v", err)
			return
		}
	}

	if err := subscribeBlockHeaders(n.smcClient.ChainReader(), n.smcClient, n.smcClient.Account()); err != nil {
		log.Errorf("Could not fetch current block number: %v", err)
		return
	}
}
