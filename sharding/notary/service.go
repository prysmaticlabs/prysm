// Package notary defines all relevant functionality for a Notary actor
// within a sharded Ethereum blockchain.
package notary

import (
	"fmt"
	"github.com/ethereum/go-ethereum/sharding"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/p2p/messages"
	"github.com/ethereum/go-ethereum/sharding/params"
)

// Notary holds functionality required to run a collation notary
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Notary struct {
	config    *params.Config
	smcClient *mainchain.SMCClient
	p2p       *p2p.Server
	dbService *database.ShardDB
}

// NewNotary creates a new notary instance.
func NewNotary(config *params.Config, smcClient *mainchain.SMCClient, p2p *p2p.Server, dbService *database.ShardDB) (*Notary, error) {
	return &Notary{config, smcClient, p2p, dbService}, nil
}

// Start the main routine for a notary.
func (n *Notary) Start() {
	log.Info("Starting notary service")
	go n.notarizeCollations()
}

// Stop the main loop for notarizing collations.
func (n *Notary) Stop() error {
	log.Info("Stopping notary service")
	return nil
}

// notarizeCollations checks incoming block headers and determines if
// we are an eligible notary for collations.
func (n *Notary) notarizeCollations() {

	// TODO: handle this better through goroutines. Right now, these methods
	// are blocking.
	if n.smcClient.DepositFlag() {
		if err := joinNotaryPool(n.smcClient, n.smcClient, n.config); err != nil {
			log.Error(fmt.Sprintf("Could not fetch current block number: %v", err))
			return
		}
	}

	if err := subscribeBlockHeaders(n.smcClient.ChainReader(), n.smcClient, n.smcClient.Account()); err != nil {
		log.Error(fmt.Sprintf("Could not fetch current block number: %v", err))
		return
	}
}

func (n *Notary) requestCollation() (*sharding.Collation, error) {

	n.p2p.Start()

	feed := n.p2p.Feed(messages.CollationBodyRequest{})
	ch := make(chan p2p.Message, 10)
	sub := feed.Subscribe(ch)
	msg := <-ch

	if _, ok := msg.Data.(messages.CollationBodyResponse); !ok {
		return nil, fmt.Errorf("response is not the correct data type")
	}

	log.Info("Collation body received: %v", msg.Data)
	return nil, nil
}
