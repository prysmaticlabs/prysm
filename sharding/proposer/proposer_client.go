package proposer

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/client"
	cli "gopkg.in/urfave/cli.v1"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system
type Proposer interface {
	Start() error
}

type proposer struct {
	client client.Client
}

// NewProposer creates a struct instance
func NewProposer(ctx *cli.Context) Proposer {
	return &proposer{
		client: client.NewClient(ctx),
	}
}

// Start the main entry point for proposing collations
func (p *proposer) Start() error {
	log.Info("Starting proposer client")
	err := p.client.Start()
	if err != nil {
		return err
	}
	defer p.client.Close()

	// TODO: Propose collations

	return nil
}
