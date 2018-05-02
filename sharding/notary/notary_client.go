package notary

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/client"
	cli "gopkg.in/urfave/cli.v1"
)

// Notary runnable client
type Notary interface {
	// Start the main routine for a notary
	Start() error
}

type notary struct {
	client client.Client
}

// NewNotary creates a new notary instance
func NewNotary(ctx *cli.Context) Notary {
	return &notary{
		client: client.NewClient(ctx),
	}
}

// Start the main routine for a notary
func (c *notary) Start() error {
	log.Info("Starting notary client")
	err := c.client.Start()
	if err != nil {
		return err
	}
	defer c.client.Close()

	if err := joinNotaryPool(c.client); err != nil {
		return err
	}

	return subscribeBlockHeaders(c.client)
}
