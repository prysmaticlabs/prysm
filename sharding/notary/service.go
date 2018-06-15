// Package notary defines all relevant functionality for a Notary actor
// within a sharded Ethereum blockchain.
package notary

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
)

// Notary holds functionality required to run a collation notary
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Notary struct {
	config       *params.Config
	smcClient    *mainchain.SMCClient
	p2p          *p2p.Server
	shardChainDb ethdb.Database
}

// NewNotary creates a new notary instance.
func NewNotary(config *params.Config, smcClient *mainchain.SMCClient, p2p *p2p.Server, shardChainDb ethdb.Database) (*Notary, error) {
	return &Notary{config, smcClient, p2p, shardChainDb}, nil
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
		if err := joinNotaryPool(n.smcClient, n.config); err != nil {
			log.Error(fmt.Sprintf("Could not fetch current block number: %v", err))
			return
		}
	}

	headerChan := make(chan *types.Header, 16)

	_, err := n.smcClient.ChainReader().SubscribeNewHead(context.Background(), headerChan)
	if err != nil {
		log.Error(fmt.Sprintf("unable to subscribe to incoming headers. %v", err))
		return
	}

	log.Info("Listening for new headers...")

	for {
		// TODO: Error handling for getting disconnected from the client.
		head := <-headerChan
		// Query the current state to see if we are an eligible notary.
		log.Info(fmt.Sprintf("Received new header: %v", head.Number.String()))

		// Check if we are in the notary pool before checking if we are an eligible notary.
		v, err := isAccountInNotaryPool(n.smcClient)
		if err != nil {
			log.Error(fmt.Sprintf("unable to verify client in notary pool. %v", err))
			return
		}

		if v {
			if err := checkSMCForNotary(n.smcClient, head); err != nil {
				log.Error(fmt.Sprintf("unable to watch shards. %v", err))
				return
			}
		}
	}
}
