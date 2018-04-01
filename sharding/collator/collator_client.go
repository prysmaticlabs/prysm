// Package collator holds all of the functionality to run as a collator in a sharded system.
package collator

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/client"
	cli "gopkg.in/urfave/cli.v1"
)

// Collator runnable client.
type Collator interface {
	// Start the main routine for a collator.
	Start() error
}

type collator struct {
	client client.Client
}

// NewCollator creates a new collator instance.
func NewCollator(ctx *cli.Context) Collator {
	return &collator{
		client: client.NewClient(ctx),
	}
}

// Start the main routine for a collator.
func (c *collator) Start() error {
	log.Info("Starting collator client")
	err := c.client.Start()
	if err != nil {
		return err
	}
	defer c.client.Close()

	if err := joinCollatorPool(c.client); err != nil {
		return err
	}

	return subscribeBlockHeaders(c.client)
}
