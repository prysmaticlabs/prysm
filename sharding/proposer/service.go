// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	client   *mainchain.SMCClient
	shardp2p sharding.ShardP2P
	txpool   sharding.TXPool
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a shardp2p network,
// and a shard transaction pool.
func NewProposer(client *mainchain.SMCClient, shardp2p sharding.ShardP2P, txpool sharding.TXPool) (*Proposer, error) {
	// Initializes a  directory persistent db.
	return &Proposer{client, shardp2p, txpool}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() error {
	log.Info("Starting proposer service")
	// TODO: Propose collations.
	return nil
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Info("Stopping proposer service")
	return nil
}
