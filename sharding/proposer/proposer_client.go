package proposer

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/client"
	cli "gopkg.in/urfave/cli.v1"
)

type Proposer interface {
	Start() error
}

type proposer struct {
	client client.Client
}

func NewProposer(ctx *cli.Context) Proposer {
	return &proposer{
		client: client.NewClient(ctx),
	}
}

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
