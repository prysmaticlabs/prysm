// Package attester defines all relevant functionality for a Attester actor
// within a sharded Ethereum blockchain.
package attester

import (
	"context"

	gethTypes "github.com/ethereum/go-ethereum/core/types"
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
	reader    mainchain.Reader
	ctx       context.Context
	cancel    context.CancelFunc
	headerBuf chan *gethTypes.Header
}

// NewAttester creates a new attester instance.
func NewAttester(config *params.Config, smcClient *mainchain.SMCClient, p2p *p2p.Server, dbService *database.DB) (*Attester, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Attester{
		config:    config,
		smcClient: smcClient,
		p2p:       p2p,
		dbService: dbService,
		ctx:       ctx,
		cancel:    cancel,
		headerBuf: make(chan *gethTypes.Header, 16),
	}, nil
}

// Start the main routine for a attester.
func (a *Attester) Start() {
	log.Info("Starting attester service")
	go a.run(a.ctx.Done())
}

// Stop the main loop for notarizing collations.
func (a *Attester) Stop() error {
	log.Info("Stopping attester service")
	return nil
}

// notarizeCollations checks incoming block headers and determines if
// we are an eligible attester for collations.
func (a *Attester) run(done <-chan struct{}) {

	headerSub, err := a.reader.SubscribeNewHead(context.Background(), a.headerBuf)
	if err != nil {
		log.Errorf("Could not subscribe to new head: %v", err)
	}
	defer headerSub.Unsubscribe()

	if a.smcClient.DepositFlag() {
		if err := joinAttesterPool(a.smcClient, a.smcClient); err != nil {
			log.Errorf("Could not fetch current block number: %v", err)
			return
		}
	}

	for {
		select {
		case <-done:
			log.Debug("Attester context closed, exiting goroutine")
			return

		// checks incoming block headers and determines if we are an eligible attester for collations.
		case header := <-a.headerBuf:
			log.WithFields(logrus.Fields{
				"number": header.Number.String(),
			}).Info("Received new header")

			// Check if we are in the attester pool before checking if we are an eligible attester.
			v, err := isAccountInAttesterPool(a.smcClient, a.smcClient.Account())
			if err != nil {
				log.Errorf("unable to verify client in attester pool. %v", err)
			}

			if v {
				if err := checkSMCForAttester(a.smcClient, a.smcClient.Account()); err != nil {
					log.Errorf("unable to watch shards. %v", err)
				}
			}
		}
	}

}
