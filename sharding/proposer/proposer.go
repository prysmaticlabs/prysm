package proposer

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/node"
	"github.com/ethereum/go-ethereum/swarm/api/client"
	cli "gopkg.in/urfave/cli.v1"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	client client.Client
}

// NewProposer creates a struct instance. It is initialized and
// registered as a service upon start of a sharding node.
// Has access to the public methods of this node.
func NewProposer(ctx *cli.Context, node node.Node) *Proposer {
	return &Proposer{}
}

// Start the main entry point for proposing collations.
func (p *Proposer) Start() error {
	log.Info("Starting proposer client")
	// TODO: Propose collations.
	return nil
}
