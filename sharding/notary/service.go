package notary

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	cli "gopkg.in/urfave/cli.v1"
)

// Notary holds functionality required to run a collation notary
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Notary struct {
	node sharding.Node
}

// NewNotary creates a new notary instance.
func NewNotary(ctx *cli.Context, node sharding.Node) *Notary {
	return &Notary{node}
}

// Start the main routine for a notary.
func (n *Notary) Start() error {
	log.Info("Starting notary client")
	// err := c.client.Start()
	// if err != nil {
	// 	return err
	// }
	// defer c.client.Close()

	// if c.client.DepositFlagSet() {
	// 	if err := joinNotaryPool(c.client); err != nil {
	// 		return err
	// 	}
	// }

	// return subscribeBlockHeaders(c.client)
	return nil
}
