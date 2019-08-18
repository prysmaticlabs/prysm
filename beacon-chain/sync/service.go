package sync

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared"
)

var _ = shared.Service(&RegularSync{})

// RegularSync service is responsible for handling all run time p2p related operations as the
// main entry point for network messages.
type RegularSync struct {
	ctx        context.Context
	p2p        p2p.P2P
	db         db.Database
	operations operations.Service
}

// Start the regular sync service by initializing all of the p2p sync handlers.
func (r *RegularSync) Start() {
	r.registerRPCHandlers()
	r.registerSubscribers()
}

// Stop the regular sync service.
func (r *RegularSync) Stop() error {
	return nil
}

// Status of the currently running regular sync service.
func (r *RegularSync) Status() error {
	return nil
}
